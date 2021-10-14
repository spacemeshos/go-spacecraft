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
	v1beta1Type "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	v1beta1 "k8s.io/client-go/kubernetes/typed/policy/v1beta1"

	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

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

func (k8s *Kubernetes) GetExternalPort(serviceId string, portName string) (string, error) {
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

			res = strings.SplitAfter(str, "\",\"identity\":\"")

			if len(res) >= 2 {
				res = strings.SplitAfter(res[1], "\"")
				resFinal := strings.TrimSuffix(res[0], "\"")

				return resFinal, nil
			} else {
				fmt.Println(podName + ": identity not found. Re-fetching logs. \n " + str)
			}
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

func (k8s *Kubernetes) GetPVCs() ([]string, error) {
	pvcs, err := k8s.Client.CoreV1().PersistentVolumeClaims("default").List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return []string{}, err
	}

	volumes := []string{}

	for _, pvc := range pvcs.Items {
		volumes = append(volumes, pvc.Spec.VolumeName)
	}

	return volumes, nil
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

func (k8s *Kubernetes) DisablePodRescheduling() error {
	client, err := v1beta1.NewForConfig(k8s.RestConfig)

	if err != nil {
		return err
	}

	pdb := client.PodDisruptionBudgets("default")

	pdb_config := &v1beta1Type.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pdb",
		},
		Spec: v1beta1Type.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{IntVal: 0},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"restart": "false",
				},
			},
		},
	}

	_, err = pdb.Create(context.TODO(), pdb_config, metav1.CreateOptions{})

	if err != nil {
		return err
	}

	return nil
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

	privateKey, _ := crypto.GenerateKey()
	privateKeyBytes := crypto.FromECDSA(privateKey)
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	compressedPubkey := crypto.CompressPubkey(publicKeyECDSA)

	privateKeyHex := hexutil.Encode(privateKeyBytes)
	publicKeyHex := hexutil.Encode(compressedPubkey)

	fmt.Println("creating miner-" + minerNumber + " coinbase secret")

	secretsClient := k8s.Client.CoreV1().Secrets("default")
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "miner-" + minerNumber + "-coinbase",
		},
		StringData: map[string]string{
			"privateKey": privateKeyHex,
			"publicKey":  publicKeyHex,
		},
	}
	_, err = secretsClient.Create(context.Background(), secret, metav1.CreateOptions{})

	if err != nil {
		channel.Err <- err
		return
	}

	command := []string{
		"/bin/go-spacemesh",
		"--test-mode",
		fmt.Sprintf("--listen=/ip4/0.0.0.0/tcp/%s", bindPortStr),
		"--json-server=true",
		"--config=/etc/config/config.json",
		"-d=/root/data/node",
		"--smeshing-coinbase=" + publicKeyHex[2:],
	}

	if config.EnableJsonAPI == true {
		command = append(command, "--json-port=7000")
	}

	command = append(command, "--grpc-port=6000")

	if config.DeployPyroscope == true {
		pyroscopeURL, err := k8s.GetPyroscopeURL()
		if err != nil {
			channel.Err <- err
			return
		}

		command = append(command, "--pprof-server")

		// when pyroscope scaling is done remove this if condition
		if minerNumber == "10" || minerNumber == "20" {
			command = append(command, "--profiler-url=http://"+pyroscopeURL)
			command = append(command, "--profiler-name=miner-"+minerNumber)
		}
	}

	if config.Metrics == true {
		command = append(command, "--metrics")
		command = append(command, "--metrics-port=1010")
	}

	command = append(command, "; sleep 100000000")

	envs := []apiv1.EnvVar{}

	if config.EnableGoDebug == true {
		envs = append(envs, apiv1.EnvVar{
			Name:  "GODEBUG",
			Value: "gctrace=1,scavtrace=1,gcpacertrace=1",
		})
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "miner-" + minerNumber,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "miner-" + minerNumber,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name":    "miner-" + minerNumber,
						"restart": "false",
						"app":     "miner",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "miner",
							Image:   config.GoSmImage,
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{strings.Join(command[:], " ")},
							Env:     envs,
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
								{
									ContainerPort: 6060,
								},
								{
									ContainerPort: 1010,
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

	if err != nil {
		channel.Err <- err
		return
	}

	fmt.Println("creating miner-" + minerNumber + " service")

	ports := []corev1.ServicePort{
		corev1.ServicePort{Name: "grpcport", Port: 6000, TargetPort: intstr.FromInt(6000)},
	}

	if config.EnableJsonAPI == true {
		ports = append(ports, corev1.ServicePort{Name: "jsonport", Port: 7000, TargetPort: intstr.FromInt(7000)})
	}

	if config.DeployPyroscope == true {
		ports = append(ports, corev1.ServicePort{Name: "pprof", Port: 6060, TargetPort: intstr.FromInt(6060)})
	}

	_, err = k8s.Client.CoreV1().Services("default").Create(context.TODO(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "miner-" + minerNumber,
			Labels: map[string]string{
				"name": "miner-" + minerNumber,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
			Selector: map[string]string{
				"name": "miner-" + minerNumber,
			},
			Type: apiv1.ServiceTypeNodePort,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		channel.Err <- err
		return
	}

	if config.Metrics == true {
		ports = []corev1.ServicePort{
			corev1.ServicePort{Name: "metrics", Port: 1010, TargetPort: intstr.FromInt(1010)},
		}

		_, err = k8s.Client.CoreV1().Services("default").Create(context.TODO(), &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "miner-" + minerNumber + "-metric",
				Labels: map[string]string{
					"name": "miner-" + minerNumber,
					"app":  "miner",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: ports,
				Selector: map[string]string{
					"name": "miner-" + minerNumber,
				},
				Type: apiv1.ServiceTypeClusterIP,
			},
		}, metav1.CreateOptions{})

		if err != nil {
			channel.Err <- err
			return
		}
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

	apiPort := ""

	grpcport, err := k8s.GetExternalPort("miner-"+minerNumber, "grpcport")
	apiPort = grpcport
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
		fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s", externalIP, bindPortStr, nodeId),
		externalIP + ":" + apiPort,
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
		"/bin/poet",
		"--restlisten=0.0.0.0:5000",
		"--initialduration=" + initialDuration,
		"--jsonlog",
		"--configfile=/etc/config/config.conf",
		"; sleep 100000000",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "poet-" + poetNumber,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "poet-" + poetNumber,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name":    "poet-" + poetNumber,
						"restart": "false",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:    "poet",
							Image:   config.PoetImage,
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{strings.Join(command[:], " ")},
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
				"name": "poet-" + poetNumber,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				corev1.ServicePort{Name: "restport", Port: 5000, TargetPort: intstr.FromInt(5000)},
			},
			Selector: map[string]string{
				"name": "poet-" + poetNumber,
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

	port, err := k8s.GetExternalPort("poet-"+poetNumber, "restport")

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

func (k8s *Kubernetes) NextMinerName() (string, error) {
	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)
	deployments, err := deploymentClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	latest := 0

	for _, deployment := range deployments.Items {
		if strings.Contains(deployment.Name, "miner-") {
			s := strings.Split(deployment.Name, "-")
			i, err := strconv.Atoi(s[1])

			if err != nil {
				return "", err
			}

			if i > latest {
				latest = i
			}
		}
	}

	return strconv.Itoa(latest + 1), nil
}

func (k8s *Kubernetes) GetMiners() ([]string, error) {
	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)
	deployments, err := deploymentClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return []string{}, err
	}

	miners := []string{}

	for _, deployment := range deployments.Items {
		if strings.Contains(deployment.Name, "miner-") {
			miners = append(miners, deployment.Name)
		}
	}

	return miners, nil
}

func (k8s *Kubernetes) UpdateImageOfMiners(name string) error {
	fmt.Println("updating image of " + name)
	deploymentClient := k8s.Client.AppsV1().Deployments(apiv1.NamespaceDefault)
	deployment, err := deploymentClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	deployment.Spec.Template.Spec.Containers[0].Image = config.GoSmImage

	_, err = deploymentClient.Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	for range time.Tick(5 * time.Second) {
		deployment, err = deploymentClient.Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		fmt.Println("waiting for " + name + " to start")

		if deployment.Status.ReadyReplicas == 1 {
			break
		}
	}

	fmt.Println("updated image of " + name)

	return nil
}

func (k8s *Kubernetes) GetExternalIP() (string, error) {
	nodes, err := k8s.Client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	node := nodes.Items[0]

	if err != nil {
		return "", err
	}

	for _, address := range node.Status.Addresses {
		if address.Type == apiv1.NodeExternalIP {
			return address.Address, nil
		}
	}

	return "", errors.New("public ip of cluster not found")
}

func (k8s *Kubernetes) MinerAccounts() ([]string, error) {
	secretsClient := k8s.Client.CoreV1().Secrets("default")
	secrets, err := secretsClient.List(context.Background(), metav1.ListOptions{})

	if err != nil {
		return []string{}, err
	}

	addresses := []string{}

	for _, secret := range secrets.Items {
		if val, ok := secret.Data["publicKey"]; ok {
			publicKey := string(val)
			addresses = append(addresses, publicKey[len(publicKey)-40:])
		}
	}

	return addresses, nil
}

func int32Ptr(i int32) *int32 { return &i }
