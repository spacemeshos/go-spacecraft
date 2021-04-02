package network

import (
	"github.com/spacemeshos/spacecraft/gcp"
	k8s "github.com/spacemeshos/spacecraft/k8s"
)

func DeleteMiner() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient()

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	err = kubernetes.DeleteMiner(config.MinerNumber)

	if err != nil {
		return err
	}

	return nil
}
