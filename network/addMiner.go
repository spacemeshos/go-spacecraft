package network

import (
	"io/ioutil"

	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
)

func AddMiner() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(config.NetworkName)

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}
	minerNumber := ""

	if config.MinerNumber != "" {
		minerNumber = config.MinerNumber
	} else {
		minerNumber, err = kubernetes.NextMinerName()

		if err != nil {
			return err
		}
	}

	configStr := ""

	if config.MinerGoSmConfig == "" {
		configStr, err = gcp.ReadConfig()

		if err != nil {
			return err
		}
	} else {
		buf, err := ioutil.ReadFile(config.MinerGoSmConfig)

		if err != nil {
			return err
		}

		configStr = string(buf)
	}

	minerChan := &k8s.MinerChannel{
		Err:  make(chan error),
		Done: make(chan *k8s.MinerDeploymentData),
	}

	go kubernetes.DeployMiner(false, minerNumber, configStr, "", minerChan)
	select {
	case err := <-minerChan.Err:
		return err
	case _ = <-minerChan.Done:
		return nil
	}
}
