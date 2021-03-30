package k8s

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cfg "github.com/spacemeshos/spacecraft/config"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

var config = &cfg.Config

type Kubernetes struct {
	Client     *kubernetes.Clientset
	RestConfig *restclient.Config
}

type Deployment struct {
	Name          string
	Labels        map[string]string
	ContainerName string
}

func (k8s *Kubernetes) execueCommand() string {
	return ""
}

func (k8s *Kubernetes) getExternalIpOfNode(nodeId string) (string, error) {
	node, err := k8s.Client.CoreV1().Nodes().Get(context.Background(), nodeId, metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	for _, address := range node.Status.Addresses {
		if address.Type == apiv1.NodeExternalIP {
			return address.Address, nil
		}
	}

	return "", errors.New("public ip of node not found")
}

func (k8s *Kubernetes) getExternalPort(serviceId string, portName string) (string, error) {
	svc, err := k8s.Client.CoreV1().Services("default").Get(context.TODO(), serviceId, metav1.GetOptions{})

	if err != nil {
		return "", err
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == portName {
			return strconv.FormatInt(int64(port.NodePort), 10), nil
		}
	}

	return "", errors.New("port not found")
}

func (k8s *Kubernetes) getNodeId(podName string) (string, error) {
	podLogOpts := corev1.PodLogOptions{}
	req := k8s.Client.CoreV1().Pods("default").GetLogs(podName, &podLogOpts)
	podLogs, err := req.Stream(context.Background())
	defer podLogs.Close()

	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	str := buf.String()

	res := strings.SplitAfter(str, "Local node identity >> ")
	res = strings.SplitAfter(res[1], "\"")
	resFinal := strings.TrimSuffix(res[0], "\"")

	return resFinal, nil
}

func (k8s *Kubernetes) DeployMiner(bootstrapNode bool, bootnodes []string, minerNumber string, configJSON string) (string, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "miner-" + minerNumber,
		},
		Data: map[string]string{"config.json": configJSON},
	}

	_, err := k8s.Client.CoreV1().ConfigMaps("default").Create(context.TODO(), configMap, metav1.CreateOptions{})

	if err != nil {
		return "", err
	}

	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)

	command := []string{
		"--test-mode",
		"--tcp-port=5000",
		"--acquire-port=0",
		"--grpc-port=6000",
		"--json-port=7000",
		"--json-server=true",
		"--start-mining",
		"--grpc-server",
		"--grpc-port-new=8000",
		"--coinbase=7566a5e003748be1c1a999c62fbe2610f69237f57ac3043f3213983819fe3ea5",
		"--config=/etc/config/config.json",
	}

	if bootstrapNode == false {
		command = append(command, "--bootstrap")
		command = append(command, "--bootnodes="+strings.Join(bootnodes[:], ","))
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "miner-" + minerNumber,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"miner": minerNumber,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"miner": minerNumber,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "miner",
							Image:   config.GoSmImage,
							Command: []string{"/bin/go-spacemesh"},
							Args:    command,
							Ports: []apiv1.ContainerPort{
								{
									ContainerPort: 5000,
								},
								{
									ContainerPort: 6000,
								},
								{
									ContainerPort: 7000,
								},
								{
									ContainerPort: 8000,
								},
							},
							Resources: apiv1.ResourceRequirements{
								Limits: apiv1.ResourceList{
									"cpu":    resource.MustParse(config.MinerCPU),
									"memory": resource.MustParse(config.MinerMemory + "Gi"),
								},
								Requests: apiv1.ResourceList{
									"cpu":    resource.MustParse(config.MinerCPU),
									"memory": resource.MustParse(config.MinerMemory + "Gi"),
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/config",
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "config",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "miner-" + minerNumber,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	deployment, err = deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})

	if err != nil {
		return "", err
	}

	fmt.Println("creating miner-" + minerNumber + " deployment")

	for range time.Tick(5 * time.Second) {
		deployment, err := deploymentClient.Get(context.TODO(), "miner-"+minerNumber, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		fmt.Println("waiting for miner-" + minerNumber + " deployment")

		if deployment.Status.ReadyReplicas == 1 {
			break
		}
	}

	fmt.Println("finished miner-" + minerNumber + " deployment")

	fmt.Println("creating miner-" + minerNumber + " service")

	_, err = k8s.Client.CoreV1().Services("default").Create(context.TODO(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "miner-" + minerNumber,
			Labels: map[string]string{
				"miner": minerNumber,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{Name: "tcpport", Port: 5000, TargetPort: intstr.FromInt(5000)},
				corev1.ServicePort{Name: "grpcport", Port: 6000, TargetPort: intstr.FromInt(6000)},
				corev1.ServicePort{Name: "jsonport", Port: 7000, TargetPort: intstr.FromInt(7000)},
				corev1.ServicePort{Name: "grpcportnew", Port: 8000, TargetPort: intstr.FromInt(8000)},
			},
			Selector: map[string]string{
				"miner": minerNumber,
			},
			Type: apiv1.ServiceTypeNodePort,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		return "", err
	}

	fmt.Println("created miner-" + minerNumber + " service")

	pods, err := k8s.Client.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return "", err
	}

	nodeName := ""
	podName := ""

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "miner-"+minerNumber) {
			nodeName = pod.Spec.NodeName
			podName = pod.Name
		}
	}

	externalIP, err := k8s.getExternalIpOfNode(nodeName)

	if err != nil {
		return "", err
	}

	port, err := k8s.getExternalPort("miner-"+minerNumber, "tcpport")

	if err != nil {
		return "", err
	}

	nodeId, err := k8s.getNodeId(podName)

	if err != nil {
		return "", err
	}

	return "spacemesh://" + nodeId + "@" + externalIP + ":" + port, nil
}

func (k8s *Kubernetes) DeployBootstrap() (string, error) {
	// svc, err := k8s.Client.CoreV1().Services("default").Get(context.TODO(), "demo-service-test", metav1.GetOptions{})

	// fmt.Println(err)
	// fmt.Println(svc)

	// deploymentsClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)

	// d, err := deploymentsClient.Get(context.TODO(), "demo-deployment-test", metav1.GetOptions{})
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(d)

	// pods, err := k8s.Client.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})

	// fmt.Println(pods)

	return "", nil

	// deployment := &appsv1.Deployment{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name: "demo-deployment-test",
	// 	},
	// 	Spec: appsv1.DeploymentSpec{
	// 		Replicas: int32Ptr(2),
	// 		Selector: &metav1.LabelSelector{
	// 			MatchLabels: map[string]string{
	// 				"app": "demo-test",
	// 			},
	// 		},
	// 		Template: apiv1.PodTemplateSpec{
	// 			ObjectMeta: metav1.ObjectMeta{
	// 				Labels: map[string]string{
	// 					"app": "demo-test",
	// 				},
	// 			},
	// 			Spec: apiv1.PodSpec{
	// 				Containers: []apiv1.Container{
	// 					{
	// 						Name:  "web",
	// 						Image: "nginx:1.12",
	// 						Ports: []apiv1.ContainerPort{
	// 							{
	// 								Name:          "http",
	// 								Protocol:      apiv1.ProtocolTCP,
	// 								ContainerPort: 80,
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// }

	// fmt.Println("Creating deployment...")
	// _, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Deployment Created...")

	// svc, err := k8s.Client.CoreV1().Services("default").Create(context.TODO(), &corev1.Service{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name: "demo-service-test",
	// 		Labels: map[string]string{
	// 			"app": "demo-test",
	// 		},
	// 	},
	// 	Spec: corev1.ServiceSpec{
	// 		Ports: []corev1.ServicePort{corev1.ServicePort{Name: "http", Port: 80, TargetPort: intstr.FromInt(80)}},
	// 		Selector: map[string]string{
	// 			"app": "demo-test",
	// 		},
	// 		Type: apiv1.ServiceTypeNodePort,
	// 	},
	// }, metav1.CreateOptions{})

	// fmt.Println(svc)

	// svc, err = k8s.Client.CoreV1().Services("default").Get(context.TODO(), "demo-service-test", metav1.GetOptions{})

	// fmt.Println(err)
	// fmt.Println(svc)

	// return "", nil
}

func int32Ptr(i int32) *int32 { return &i }
