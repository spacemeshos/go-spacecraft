package network

import (
	"github.com/spacemeshos/go-spacecraft/gcp"
	k8s "github.com/spacemeshos/go-spacecraft/k8s"
)

func DeployWS() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(config.NetworkName)
	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	if err = kubernetes.DeployFilebeatForWS(); err != nil {
		return err
	}

	err = kubernetes.DeployWS()

	if err != nil {
		return err
	}

	if err = kubernetes.SetupLogDeletionPolicyForWS(); err != nil {
		return err
	}

	err = kubernetes.AddToDiscovery()

	if err != nil {
		return err
	}

	return nil
}
