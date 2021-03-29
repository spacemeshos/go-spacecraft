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

	k8sClient, err := gcp.GetKubernetesClient()

	if err != nil {
		return err
	}

	kubernetes = k8s.Kubernetes{k8sClient}

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

	minerConfigJsonStr := minerConfigJson.String()

	layerDurationSec, ok := minerConfigJson.Path("main.layer-duration-sec").Data().(float64)

	if ok == false {
		return errors.New("cannot read layer-duration-sec from config file")
	}

	layersPerEpoch, ok := minerConfigJson.Path("main.layers-per-epoch").Data().(float64)

	if ok == false {
		return errors.New("cannot read layers-per-epoch from config file")
	}

	poetConfig := "duration = \"" + fmt.Sprintf("%d", int((layerDurationSec)*(layersPerEpoch))) + "s\"\nn=\"21\""

	fmt.Println(minerConfigJsonStr)
	fmt.Println(poetConfig)

	var poetInitialDurations []string
	shift := 0
	for i := 0; i < config.NumberOfPoets; i++ {
		initialduration := (genesisMinutes * 60) + int(layerDurationSec) + shift
		poetInitialDurations = append(poetInitialDurations, strconv.Itoa(initialduration)+"s")
		shift = shift + config.InitPhaseShift
	}

	fmt.Println(poetInitialDurations)

	return nil
}
