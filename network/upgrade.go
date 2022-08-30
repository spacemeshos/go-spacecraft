package network

import (
	"time"

	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
)

func Upgrade() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(config.NetworkName)
	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	miners, err := kubernetes.GetMiners()
	if err != nil {
		return err
	}

	for _, miner := range miners {
		kubernetes.UpdateImageOfMiners(miner)
		time.Sleep(time.Duration(config.RestartWaitTime) * time.Minute)
	}

	return nil
}
