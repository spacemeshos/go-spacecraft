package network

import (
	"context"
	"strings"

	"github.com/spacemeshos/go-spacecraft/gcp"
	"github.com/spacemeshos/go-spacecraft/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Delete() error {
	k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(config.NetworkName)

	if err != nil {
		return err
	}

	kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

	if !config.KeepLogsMetrics {
		volumes, err := kubernetes.GetPVCs()

		if err != nil {
			return err
		}

		err = gcp.DeleteKubernetesCluster(volumes)

		if err != nil {
			return err
		}

		err = kubernetes.DeleteELKDNSRecords()

		if err != nil {
			return err
		}
	} else {
		k8sRestConfig, k8sClient, err := gcp.GetKubernetesClient(config.NetworkName)

		if err != nil {
			return err
		}

		kubernetes := k8s.Kubernetes{Client: k8sClient, RestConfig: k8sRestConfig}

		deployments, err := kubernetes.Client.AppsV1().Deployments("default").List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			return err
		}

		for _, deployment := range deployments.Items {
			if strings.Contains(deployment.Name, "miner") || strings.Contains(deployment.Name, "poet") || strings.Contains(deployment.Name, "watch") {
				err := kubernetes.Client.AppsV1().Deployments("default").Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{})

				if err != nil {
					return err
				}
			}
		}

		err = kubernetes.Client.CoreV1().Namespaces().Delete(context.TODO(), "ws", metav1.DeleteOptions{})

		if err != nil {
			return err
		}

		err = gcp.ResizeKubernetesClusterForLogs()

		if err != nil {
			return err
		}
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
