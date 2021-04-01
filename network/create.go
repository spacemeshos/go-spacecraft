package network

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	gabs "github.com/Jeffail/gabs/v2"

	"github.com/spacemeshos/spacecraft/gcp"
	k8s "github.com/spacemeshos/spacecraft/k8s"
)

func Create() error {
	err := gcp.CreateKubernetesCluster()

	if err != nil {
		return err
	}

	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient()

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{k8sClient, k8sRestConfig}

	minerConfigBuf, err := ioutil.ReadFile(config.GoSmConfig)

	if err != nil {
		return err
	}

	minerConfigStr := string(minerConfigBuf)
	minerConfigJson, err := gabs.ParseJSON([]byte(minerConfigStr))

	genesisMinutes := 5

	genesisTime := time.Now().Add(time.Duration(genesisMinutes) * time.Minute).Format(time.RFC3339)
	minerConfigJson.SetP(genesisTime, "main.genesis-time")
	minerConfigJson.SetP(config.NumberOfMiners, "main.genesis-active-size")

	minerConfigJson.SetP(int(((float64(config.NumberOfMiners) / 100) * 60)), "hare.hare-committee-size")
	minerConfigJson.SetP(int((((float64(config.NumberOfMiners)/100)*60)/2)-1), "hare.hare-max-adversaries")

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

	for i := 0; i < config.NumberOfPoets; i++ {
		poet, err := kubernetes.DeployPoet(poetInitialDurations[i], strconv.Itoa(i+1), poetConfig)

		if err != nil {
			return err
		}

		poetRESTUrls = append(poetRESTUrls, poet)
	}

	minerConfigJson.SetP(poetRESTUrls[0], "main.poet-server")

	var miners []string
	var minerGRPCURls []string
	for i := 0; i < config.NumberOfMiners; i++ {
		if i == 0 {
			tcpurl, grpcurl, err := kubernetes.DeployMiner(true, strconv.Itoa(i+1), minerConfigJson.String())
			if err != nil {
				return err
			}

			miners = append(miners, tcpurl)
			minerGRPCURls = append(minerGRPCURls, grpcurl)
		} else if i < config.BootnodeAmount {
			minerConfigJson.SetP(true, "p2p.swarm.bootstrap")
			minerConfigJson.SetP(miners[0:1], "p2p.swarm.bootnodes")

			tcpurl, grpcurl, err := kubernetes.DeployMiner(false, strconv.Itoa(i+1), minerConfigJson.String())
			if err != nil {
				return err
			}

			miners = append(miners, tcpurl)
			minerGRPCURls = append(minerGRPCURls, grpcurl)
		} else {
			minerConfigJson.SetP(true, "p2p.swarm.bootstrap")
			minerConfigJson.SetP(miners[0:config.BootnodeAmount], "p2p.swarm.bootnodes")

			tcpurl, grpcurl, err := kubernetes.DeployMiner(false, strconv.Itoa(i+1), minerConfigJson.String())
			if err != nil {
				return err
			}

			miners = append(miners, tcpurl)
			minerGRPCURls = append(minerGRPCURls, grpcurl)

		}
	}

	gateways := minerGRPCURls[:len(minerGRPCURls)-config.PoetGatewayAmount]

	for i := 0; i < config.NumberOfPoets; i++ {
		poetRESTUrl := poetRESTUrls[i]
		postBody, _ := json.Marshal(map[string][]string{
			"gatewayAddresses": gateways,
		})
		responseBody := bytes.NewBuffer(postBody)
		_, err := http.Post("http://"+poetRESTUrl+"/v1/start", "application/json", responseBody)

		if err != nil {
			return err
		}
	}

	return nil
}
