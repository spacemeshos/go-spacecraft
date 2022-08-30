package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func (k8s *Kubernetes) DeployPyroscope() error {
	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)

	fmt.Println("creating pyroscope deployment")

	command := []string{
		"pyroscope",
		"server",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pyroscope",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "pyroscope",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": "pyroscope",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "pyroscope",
							Image:   config.PyroscopeImage,
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{strings.Join(command[:], " ")},
							Ports: []apiv1.ContainerPort{
								{
									ContainerPort: 4040,
								},
							},
							Resources: apiv1.ResourceRequirements{
								Limits: apiv1.ResourceList{
									"cpu":    resource.MustParse(config.PyroscopeCPU),
									"memory": resource.MustParse(config.PyroscopeMemory + "Gi"),
								},
								Requests: apiv1.ResourceList{
									"cpu":    resource.MustParse(config.PyroscopeCPU),
									"memory": resource.MustParse(config.PyroscopeMemory + "Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	for range time.Tick(5 * time.Second) {
		deployment, err := deploymentClient.Get(context.TODO(), "pyroscope", metav1.GetOptions{})
		if err != nil {
			return err
		}

		fmt.Println("waiting for pyroscope deployment")

		if deployment.Status.ReadyReplicas == 1 {
			break
		}
	}

	fmt.Println("finished pyroscope deployment")

	fmt.Println("creating pyroscope service")

	_, err = k8s.Client.CoreV1().Services("default").Create(context.TODO(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pyroscope",
			Labels: map[string]string{
				"name": "pyroscope",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 4040, TargetPort: intstr.FromInt(4040)},
			},
			Selector: map[string]string{
				"name": "pyroscope",
			},
			Type: apiv1.ServiceTypeNodePort,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		return err
	}

	fmt.Println("finished creating pyroscope service")

	return nil
}

func (k8s *Kubernetes) GetPyroscopeURL() (string, error) {
	ip, err := k8s.GetExternalIP()
	if err != nil {
		return "", err
	}

	port, err := k8s.GetExternalPort("pyroscope", "http")
	if err != nil {
		return "", err
	}

	return ip + ":" + port, nil
}
