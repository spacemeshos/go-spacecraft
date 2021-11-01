package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k8s *Kubernetes) DeploySpacemeshWatch() error {
	fmt.Println("deploying spacemesh watch")

	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)

	miners, err := k8s.GetMiners()

	if err != nil {
		return err
	}

	apiURLs := []string{}

	ip, err := k8s.GetExternalIP()

	if err != nil {
		return err
	}

	for _, miner := range miners {
		port, err := k8s.GetExternalPort(miner, "grpcport")
		if err != nil {
			return err
		}

		apiURLs = append(apiURLs, ip+":"+port+"/"+miner)
	}

	command := []string{
		"/bin/spacemesh-watch",
		"--nodes=" + strings.Join(apiURLs[:], ","),
		"--network-name=" + config.NetworkName,
	}

	if config.SlackToken != "" && config.SlackChannelId != "" {
		command = append(command, "--slack-api-token="+config.SlackToken)
		command = append(command, "--slack-channel-name="+config.SlackChannelId)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "spacemesh-watch",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "spacemesh-watch",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": "spacemesh-watch",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "spacemesh-watch",
							Image:   config.SpacemeshWatchImage,
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{strings.Join(command[:], " ")},
						},
					},
				},
			},
		},
	}

	_, err = deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})

	if err != nil {
		return err
	}

	for range time.Tick(5 * time.Second) {
		deployment, err := deploymentClient.Get(context.TODO(), "spacemesh-watch", metav1.GetOptions{})
		if err != nil {
			return err
		}

		fmt.Println("waiting for spacemesh-watch deployment")

		if deployment.Status.ReadyReplicas == 1 {
			break
		}
	}

	fmt.Println("finished spacemesh-watch deployment")

	return nil
}

func (k8s *Kubernetes) DeleteSpacemeshWatch() error {
	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)

	err := deploymentClient.Delete(context.TODO(), "spacemesh-watch", metav1.DeleteOptions{})

	if err != nil {
		return err
	}

	return nil
}
