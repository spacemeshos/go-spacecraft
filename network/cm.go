package network

import (
	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
)

func DeployCM() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(config.NetworkName)

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	err = kubernetes.DeployChaosMesh()

	if err != nil {
		return err
	}

	return nil
}
