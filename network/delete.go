package network

import (
	"github.com/spacemeshos/go-spacecraft/gcp"
	"github.com/spacemeshos/go-spacecraft/k8s"
)

func Delete() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient()

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	volumes, err := kubernetes.GetPVCs()

	if err != nil {
		return err
	}

	err = gcp.DeleteKubernetesCluster(volumes)

	if err != nil {
		return err
	}

	err = kubernetes.DeleteMetricsDNSRecords()

	if err != nil {
		return err
	}

	err = kubernetes.DeleteELKDNSRecords()

	if err != nil {
		return err
	}

	err = kubernetes.DeleteWSDNSRecords()

	if err != nil {
		return err
	}

	err = kubernetes.RemoveFromDiscovery()

	if err != nil {
		return err
	}

	return nil
}
