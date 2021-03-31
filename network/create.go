package network

import (
	"errors"
	"fmt"
	"io/ioutil"
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

	var poetGateways []string

	for i := 0; i < len(poetInitialDurations); i++ {
		poet, err := kubernetes.DeployPoet(poetInitialDurations[i], strconv.Itoa(i+1), poetConfig)

		if err != nil {
			return err
		}

		poetGateways = append(poetGateways, poet)
	}

	minerConfigJson.SetP(poetGateways[0], "main.poet-server")

	var miners []string
	noOfMiners := int(config.NumberOfMiners)

	for i := 0; i < noOfMiners; i++ {
		var bootnodes []string

		if i > 0 {
			bootnodes = append(bootnodes, miners[0])
		}

		if i == 0 {

			miner, err := kubernetes.DeployMiner(true, strconv.Itoa(i+1), minerConfigJson.String())
			if err != nil {
				return err
			}

			miners = append(miners, miner)
		} else if i < 5 {
			minerConfigJson.SetP(true, "p2p.swarm.bootstrap")
			minerConfigJson.SetP([]string{miners[0]}, "p2p.swarm.bootnodes")

			miner, err := kubernetes.DeployMiner(false, strconv.Itoa(i+1), minerConfigJson.String())
			if err != nil {
				return err
			}

			miners = append(miners, miner)
		} else {
			minerConfigJson.SetP(true, "p2p.swarm.bootstrap")
			minerConfigJson.SetP([]string{miners[0], miners[1], miners[2], miners[3], miners[4]}, "p2p.swarm.bootnodes")

			miner, err := kubernetes.DeployMiner(false, strconv.Itoa(i+1), minerConfigJson.String())
			if err != nil {
				return err
			}

			miners = append(miners, miner)
		}
	}

	return nil
}
