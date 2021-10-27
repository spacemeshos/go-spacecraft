package network

import (
	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
)

func DeployWS() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient()

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	err = kubernetes.DeployWS()

	if err != nil {
		return err
	}

	err = kubernetes.AddToDiscovery()

	if err != nil {
		return err
	}

	return nil
}
