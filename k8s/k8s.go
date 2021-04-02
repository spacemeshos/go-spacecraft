package k8s

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
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
	Client      *kubernetes.Clientset
	RestConfig  *restclient.Config
	CurrentNode int
	mu          sync.Mutex
}

type MinerDeploymentData struct {
	TcpURL  string
	GrpcURL string
}

type PoetDeploymentData struct {
	RestURL string
}

type MinerChannel struct {
	Err  chan error
	Done chan *MinerDeploymentData
}

type PoetChannel struct {
	Err  chan error
	Done chan *PoetDeploymentData
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
	for range time.Tick(5 * time.Second) {
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

		if len(res) >= 2 {
			res = strings.SplitAfter(res[1], "\"")
			resFinal := strings.TrimSuffix(res[0], "\"")

			return resFinal, nil
		} else {
			fmt.Println(podName + " logs not found")
		}
	}

	return "", nil
}

func (k8s *Kubernetes) createPVC(name string, size string) error {
	fs := apiv1.PersistentVolumeFilesystem

	createOpts := &apiv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1.PersistentVolumeClaimSpec{
			AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.ReadWriteOnce},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceName(apiv1.ResourceStorage): resource.MustParse(size + "Gi"),
				},
			},
			VolumeMode: &fs,
		},
		Status: apiv1.PersistentVolumeClaimStatus{
			Phase:       apiv1.ClaimBound,
			AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.ReadWriteOnce},
			Capacity: apiv1.ResourceList{
				apiv1.ResourceName(apiv1.ResourceStorage): resource.MustParse(size + "Gi"),
			},
		},
	}

	_, err := k8s.Client.CoreV1().PersistentVolumeClaims("default").Create(context.TODO(), createOpts, metav1.CreateOptions{})

	if err != nil {
		return err
	}

	return nil
}

func (k8s *Kubernetes) getDeploymentPodAndNode(name string) (string, string, error) {
	pods, err := k8s.Client.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return "", "", err
	}

	nodeName := ""
	podName := ""

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, name) {
			nodeName = pod.Spec.NodeName
			podName = pod.Name

			return nodeName, podName, nil
		}
	}

	return "", "", errors.New("pod not found")
}

func (k8s *Kubernetes) getDeploymentClusterIP(name string) (string, error) {
	pods, err := k8s.Client.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return "", err
	}

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, name) {
			return pod.Status.PodIP, nil
		}
	}

	return "", errors.New("pod not found")
}

func (k8s *Kubernetes) NextNode() (string, error) {
	nodes, err := k8s.Client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})

	if err != nil {
		return "", err
	}

	k8s.mu.Lock()
	defer k8s.mu.Unlock()

	if k8s.CurrentNode >= len(nodes.Items) {
		k8s.CurrentNode = 0
	}

	node := nodes.Items[k8s.CurrentNode]

	k8s.CurrentNode += 1

	return node.Name, nil
}

func (k8s *Kubernetes) DeployMiner(bootstrapNode bool, minerNumber string, configJSON string, selectedNode string, channel *MinerChannel) {
	fmt.Println("creating miner-" + minerNumber + " pvc")

	err := k8s.createPVC("miner-"+minerNumber, config.MinerDiskSize)

	if err != nil {
		channel.Err <- err
		return
	}

	fmt.Println("created miner-" + minerNumber + " pvc")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "miner-" + minerNumber,
		},
		Data: map[string]string{"config.json": configJSON},
	}

	_, err = k8s.Client.CoreV1().ConfigMaps("default").Create(context.TODO(), configMap, metav1.CreateOptions{})

	if err != nil {
		channel.Err <- err
		return
	}

	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)

	minerNumberInt, _ := strconv.Atoi(minerNumber)
	bindPort := int32(minerNumberInt + 5000)
	bindPortStr := strconv.Itoa(int(bindPort))

	command := []string{
		"--test-mode",
		"--tcp-port=" + bindPortStr,
		"--acquire-port=0",
		"--grpc-port=6000",
		"--json-port=7000",
		"--json-server=true",
		"--start-mining",
		"--grpc-server",
		"--grpc-port-new=8000",
		"--coinbase=7566a5e003748be1c1a999c62fbe2610f69237f57ac3043f3213983819fe3ea5",
		"--config=/etc/config/config.json",
		"--post-datadir=/root/data/post",
		"-d=/root/data/node",
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
									ContainerPort: bindPort,
									Protocol:      corev1.ProtocolTCP,
									HostPort:      bindPort,
								},
								{
									ContainerPort: bindPort,
									Protocol:      corev1.ProtocolUDP,
									HostPort:      bindPort,
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
								{
									Name:      "data",
									MountPath: "/root/data",
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
						{
							Name: "data",
							VolumeSource: apiv1.VolumeSource{
								PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
									ClaimName: "miner-" + minerNumber,
								},
							},
						},
					},
				},
			},
		},
	}

	nodeSelector := map[string]string{}

	if selectedNode != "" {
		nodeSelector["kubernetes.io/hostname"] = selectedNode
		deployment.Spec.Template.Spec.NodeSelector = nodeSelector
	}

	deployment, err = deploymentClient.Create(context.TODO(), deployment, metav1.CreateOptions{})

	if err != nil {
		channel.Err <- err
		return
	}

	fmt.Println("creating miner-" + minerNumber + " deployment")

	for range time.Tick(5 * time.Second) {
		deployment, err := deploymentClient.Get(context.TODO(), "miner-"+minerNumber, metav1.GetOptions{})
		if err != nil {
			channel.Err <- err
			return
		}

		fmt.Println("waiting for miner-" + minerNumber + " deployment")

		if deployment.Status.ReadyReplicas == 1 {
			break
		}
	}

	fmt.Println("finished miner-" + minerNumber + " deployment")

	nodeName, podName, err := k8s.getDeploymentPodAndNode("miner-" + minerNumber)

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
				// corev1.ServicePort{
				// 	Name:       "tcpport",
				// 	Port:       bindPort,
				// 	TargetPort: intstr.FromInt(int(bindPort)),
				// 	Protocol:   corev1.ProtocolTCP,
				// 	//NodePort:   30000 + int32(mn),
				// },
				// corev1.ServicePort{
				// 	Name:       "udpport",
				// 	Port:       bindPort,
				// 	TargetPort: intstr.FromInt(int(bindPort)),
				// 	Protocol:   corev1.ProtocolUDP,
				// 	//NodePort:   30000 + int32(mn),
				// },
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
		channel.Err <- err
		return
	}

	fmt.Println("created miner-" + minerNumber + " service")

	if err != nil {
		channel.Err <- err
		return
	}

	externalIP, err := k8s.getExternalIpOfNode(nodeName)

	if err != nil {
		channel.Err <- err
		return
	}

	grpcport, err := k8s.getExternalPort("miner-"+minerNumber, "grpcport")

	if err != nil {
		channel.Err <- err
		return
	}

	nodeId, err := k8s.getNodeId(podName)

	if err != nil {
		channel.Err <- err
		return
	}
	channel.Done <- &MinerDeploymentData{
		"spacemesh://" + nodeId + "@" + externalIP + ":" + bindPortStr,
		externalIP + ":" + grpcport,
	}
}

func (k8s *Kubernetes) DeployPoet(initialDuration string, poetNumber string, configFile string, channel *PoetChannel) {

	fmt.Println("creating poet-" + poetNumber + " pvc")

	err := k8s.createPVC("poet-"+poetNumber, config.MinerDiskSize)

	if err != nil {
		channel.Err <- err
		return
	}

	fmt.Println("created poet-" + poetNumber + " pvc")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "poet-" + poetNumber,
		},
		Data: map[string]string{"config.conf": configFile},
	}

	_, err = k8s.Client.CoreV1().ConfigMaps("default").Create(context.TODO(), configMap, metav1.CreateOptions{})

	if err != nil {
		channel.Err <- err
		return
	}

	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)

	command := []string{
		"--restlisten=0.0.0.0:5000",
		"--initialduration=" + initialDuration,
		"--jsonlog",
		"--configfile=/etc/config/config.conf",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "poet-" + poetNumber,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"poet": poetNumber,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"poet": poetNumber,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "poet",
							Image:   config.PoetImage,
							Command: []string{"/bin/poet"},
							Args:    command,
							Ports: []apiv1.ContainerPort{
								{
									ContainerPort: 5000,
								},
							},
							Resources: apiv1.ResourceRequirements{
								Limits: apiv1.ResourceList{
									"cpu":    resource.MustParse(config.PoetCPU),
									"memory": resource.MustParse(config.PoetMemory + "Gi"),
								},
								Requests: apiv1.ResourceList{
									"cpu":    resource.MustParse(config.PoetCPU),
									"memory": resource.MustParse(config.PoetMemory + "Gi"),
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/config",
								},
								{
									Name:      "data",
									MountPath: "/root/.poet",
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
										Name: "poet-" + poetNumber,
									},
								},
							},
						},
						{
							Name: "data",
							VolumeSource: apiv1.VolumeSource{
								PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
									ClaimName: "poet-" + poetNumber,
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
		channel.Err <- err
		return
	}

	fmt.Println("creating poet-" + poetNumber + " deployment")

	for range time.Tick(5 * time.Second) {
		deployment, err := deploymentClient.Get(context.TODO(), "poet-"+poetNumber, metav1.GetOptions{})
		if err != nil {
			channel.Err <- err
			return
		}

		fmt.Println("waiting for poet-" + poetNumber + " deployment")

		if deployment.Status.ReadyReplicas == 1 {
			break
		}
	}

	fmt.Println("finished poet-" + poetNumber + " deployment")

	fmt.Println("creating poet-" + poetNumber + " service")

	_, err = k8s.Client.CoreV1().Services("default").Create(context.TODO(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "poet-" + poetNumber,
			Labels: map[string]string{
				"poet": poetNumber,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{Name: "restport", Port: 5000, TargetPort: intstr.FromInt(5000)},
			},
			Selector: map[string]string{
				"poet": poetNumber,
			},
			Type: apiv1.ServiceTypeNodePort,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		channel.Err <- err
		return
	}

	fmt.Println("created poet-" + poetNumber + " service")

	nodeName, _, err := k8s.getDeploymentPodAndNode("poet-" + poetNumber)

	externalIP, err := k8s.getExternalIpOfNode(nodeName)

	if err != nil {
		channel.Err <- err
		return
	}

	port, err := k8s.getExternalPort("poet-"+poetNumber, "restport")

	if err != nil {
		channel.Err <- err
		return
	}

	channel.Done <- &PoetDeploymentData{externalIP + ":" + port}
}

func (k8s *Kubernetes) DeleteMiner(minerNumber string) error {
	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)

	err := deploymentClient.Delete(context.TODO(), "miner-"+minerNumber, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func int32Ptr(i int32) *int32 { return &i }
