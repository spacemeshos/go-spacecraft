package network

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	gabs "github.com/Jeffail/gabs/v2"
	"github.com/spacemeshos/go-spacecraft/log"

	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
)

func sanitizeYaml(yaml string) string {
	return strings.ReplaceAll(yaml, "	", "  ")
}

func Create() error {

	err := gcp.CreateKubernetesCluster()

	if err != nil {
		return err
	}

	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient()

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	if config.DeployPyroscope == true {
		if err = kubernetes.DeployPyroscope(); err != nil {
			return err
		}
	}

	if err = kubernetes.DeployELK(); err != nil {
		return err
	}

	if err = kubernetes.DisablePodRescheduling(); err != nil {
		return err
	}

	minerConfigBuf := []byte{}

	if config.Bootstrap {
		minerConfigBuf, err = ioutil.ReadFile(config.GoSmConfig)
	} else {
		minerConfigBuf, err = ioutil.ReadFile(config.MinerGoSmConfig)
	}

	if err != nil {
		return err
	}

	minerConfigStr := string(minerConfigBuf)
	minerConfigJson, err := gabs.ParseJSON([]byte(minerConfigStr))

	if err != nil {
		return err
	}

	genesisMinutes := config.GenesisDelay

	if config.Bootstrap {
		genesisTime := time.Now().Add(time.Duration(genesisMinutes) * time.Minute).Format(time.RFC3339)
		minerConfigJson.SetP(genesisTime, "main.genesis-time")
		minerConfigJson.SetP(config.NumberOfMiners, "main.genesis-active-size")

		if config.AdjustHare == true {
			//should be less than total miners
			minerConfigJson.SetP(int(((float64(config.NumberOfMiners) / 100) * 60)), "hare.hare-committee-size")
			//should be half-1 of hare committee size
			minerConfigJson.SetP(int((((float64(config.NumberOfMiners)/100)*60)/2)-1), "hare.hare-max-adversaries")
		}
	}

	layerDurationSec, ok := minerConfigJson.Path("main.layer-duration-sec").Data().(float64)

	if ok == false {
		return errors.New("cannot read layer-duration-sec from config file")
	}

	layersPerEpoch, ok := minerConfigJson.Path("main.layers-per-epoch").Data().(float64)

	if ok == false {
		return errors.New("cannot read layers-per-epoch from config file")
	}

	poetConfig := "duration=\"" + fmt.Sprintf("%d", int((layerDurationSec)*(layersPerEpoch))) + "s\"\nn=\"21\""

	var poetInitialDurations []string
	shift := 0
	for i := 0; i < config.NumberOfPoets; i++ {
		initialduration := (genesisMinutes * 60) + int(layerDurationSec) + shift
		poetInitialDurations = append(poetInitialDurations, strconv.Itoa(initialduration)+"s")
		shift = shift + config.InitPhaseShift
	}

	var poetRESTUrls []string

	poetChan := &k8s.PoetChannel{
		Err:  make(chan error),
		Done: make(chan *k8s.PoetDeploymentData),
	}

	//Deploy Poet(s)
	for i := 0; i < config.NumberOfPoets; i++ {
		go kubernetes.DeployPoet(poetInitialDurations[i], strconv.Itoa(i+1), poetConfig, poetChan)
	}

	for i := 0; i < config.NumberOfPoets; i++ {
		select {
		case err := <-poetChan.Err:
			return err
		case poet := <-poetChan.Done:
			poetRESTUrls = append(poetRESTUrls, poet.RestURL)
		}
	}

	//assign poets to miners in round robin fashion
	currentPoet := 0
	nextPoet := func() string {
		if currentPoet >= len(poetRESTUrls) {
			currentPoet = 0
		}

		currentPoet += 1

		return poetRESTUrls[currentPoet-1]
	}

	var miners []string
	var minerGRPCURls []string

	minerChan := &k8s.MinerChannel{
		Err:  make(chan error),
		Done: make(chan *k8s.MinerDeploymentData),
	}

	//Deploy Bootstrap
	if config.Bootstrap {
		nextNode, err := kubernetes.NextNode()
		if err != nil {
			return err
		}
		minerConfigJson.SetP(nextPoet(), "main.poet-server")
		go kubernetes.DeployMiner(true, strconv.Itoa(1), minerConfigJson.String(), nextNode, minerChan)
		select {
		case err := <-minerChan.Err:
			return err
		case miner := <-minerChan.Done:
			miners = append(miners, miner.TcpURL)
			minerGRPCURls = append(minerGRPCURls, miner.GrpcURL)
		}
	}

	start, end := 0, 0

	//Deploy bootnodes
	minerConfigJson.SetP(true, "p2p.swarm.bootstrap")
	if config.Bootstrap {
		minerConfigJson.SetP(miners[0:1], "p2p.swarm.bootnodes")
		start = 1
		end = config.BootnodeAmount
	} else {
		start = 0
		end = config.BootnodeAmount - 1
	}

	for i := start; i <= end; i++ {
		nextNode, err := kubernetes.NextNode()
		if err != nil {
			return err
		}
		minerConfigJson.SetP(nextPoet(), "main.poet-server")
		go kubernetes.DeployMiner(true, strconv.Itoa(i+1), minerConfigJson.String(), nextNode, minerChan)
	}

	for i := start; i <= end; i++ {
		select {
		case err := <-minerChan.Err:
			return err
		case miner := <-minerChan.Done:
			miners = append(miners, miner.TcpURL)
			minerGRPCURls = append(minerGRPCURls, miner.GrpcURL)
		}
	}

	//Deploy remaining miners
	if config.Bootstrap {
		minerConfigJson.SetP(miners[1:config.BootnodeAmount+1], "p2p.swarm.bootnodes")
		start = config.BootnodeAmount + 1
		end = config.NumberOfMiners
	} else {
		minerConfigJson.SetP(miners[0:config.BootnodeAmount], "p2p.swarm.bootnodes")
		start = config.BootnodeAmount
		end = config.NumberOfMiners
	}

	remainingMinerNumbers := []int{}

	for i := start; i < end; i++ {
		remainingMinerNumbers = append(remainingMinerNumbers, i)
	}

	minersChunks := chunkSlice(remainingMinerNumbers, config.MaxConcurrentDeployments)

	for i := 0; i < len(minersChunks); i++ {
		start := minersChunks[i][0]
		end := minersChunks[i][len(minersChunks[i])-1]

		for i := start; i <= end; i++ {
			minerConfigJson.SetP(nextPoet(), "main.poet-server")
			go kubernetes.DeployMiner(true, strconv.Itoa(i+1), minerConfigJson.String(), "", minerChan)
		}

		for i := start; i <= end; i++ {
			select {
			case err := <-minerChan.Err:
				return err
			case miner := <-minerChan.Done:
				miners = append(miners, miner.TcpURL)
				minerGRPCURls = append(minerGRPCURls, miner.GrpcURL)
			}
		}
	}

	//Activate poet(s)
	gateways := minerGRPCURls[0:config.PoetGatewayAmount]

	for i := 0; i < config.NumberOfPoets; i++ {
		poetRESTUrl := poetRESTUrls[i]
		postBody, _ := json.Marshal(map[string][]string{
			"gatewayAddresses": gateways,
		})
		requestBody := bytes.NewBuffer(postBody)
		resp, err := http.Post("http://"+poetRESTUrl+"/v1/start", "application/json", requestBody)

		if err != nil {
			return err
		}

		if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				return err
			}

			return errors.New(string(body))
		}
	}

	err = gcp.UploadConfig(minerConfigJson.StringIndent("", "	"))
	if err != nil {
		return err
	}
	err = kubernetes.SetupLogDeletionPolicy()
	if err != nil {
		return err
	}

	kibanaURL, err := kubernetes.GetKibanaURL()

	if err != nil {
		return err
	}

	log.Info.Println("Kibana URL: http://" + kibanaURL)
	log.Info.Println("Kibana Username: elastic")
	log.Info.Println("Kibana Password: " + kubernetes.Password)

	if config.DeployPyroscope == true {
		pyroscopeURL, err := kubernetes.GetPyroscopeURL()

		if err != nil {
			return err
		}

		log.Info.Println("Pyroscope URL: http://" + pyroscopeURL)
	}

	return nil
}

func chunkSlice(slice []int, chunkSize int) [][]int {
	var chunks [][]int
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond
		// slice capacity
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
