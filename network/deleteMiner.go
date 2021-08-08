package network

import (
	"errors"

	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
)

func DeleteMiner() error {

	if config.MinerNumber == "" {
		return errors.New("please provide miner number to delete")
	}

	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient()

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	err = kubernetes.DeleteMiner(config.MinerNumber)

	if err != nil {
		return err
	}

	if config.EnableSlackAlerts == true {
		err = kubernetes.DeleteSpacemeshWatch()

		if err != nil {
			return err
		}

		err = kubernetes.DeploySpacemeshWatch()

		if err != nil {
			return err
		}
	}

	return nil
}
