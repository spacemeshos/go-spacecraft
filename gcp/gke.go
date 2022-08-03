package gcp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	container "cloud.google.com/go/container/apiv1"
	cfg "github.com/spacemeshos/go-spacecraft/config"
	compute "google.golang.org/api/compute/v1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var config = &cfg.Config

func getClient() (*container.ClusterManagerClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if config.GCPProject == "" || config.GCPLocation == "" {
		return nil, errors.New("please provide JSON key file, project name and location for gcp authorization")
	}

	c, err := container.NewClusterManagerClient(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not authorize gcp", err)
	}

	return c, nil
}

func getCluster(networkName string) (*containerpb.Cluster, error) {
	client, err := getClient()

	if err != nil {
		return nil, err
	}

	req := &containerpb.GetClusterRequest{
		Name: "projects/" + config.GCPProject + "/locations/" + config.GCPLocation + "/clusters/" + networkName,
	}

	return client.GetCluster(context.Background(), req)
}

func GetClusters() ([]string, error) {
	client, err := getClient()

	if err != nil {
		return nil, err
	}

	req := &containerpb.ListClustersRequest{
		Parent: "projects/" + config.GCPProject + "/locations/" + config.GCPLocation,
	}

	list, err := client.ListClusters(context.Background(), req)

	if err != nil {
		return nil, err
	}

	networks := []string{}

	for _, cluster := range list.Clusters {
		networks = append(networks, cluster.Name)
	}

	return networks, nil
}

func CreateKubernetesCluster() error {
	client, err := getClient()

	if err != nil {
		return err
	}

	_, err = getCluster(config.NetworkName)

	if err == nil {
		return errors.New("cluster already exists")
	}

	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			return err
		}
	}

	minerCPUInt, _ := strconv.ParseInt(config.MinerCPU, 10, 8)
	poetCPUInt, _ := strconv.ParseInt(config.PoetCPU, 10, 8)
	esCPUInt, _ := strconv.ParseInt(config.ESCPU, 10, 8)
	kibanaCPUInt, _ := strconv.ParseInt(config.KibanaCPU, 10, 8)
	pyroscopeCPUInt, _ := strconv.ParseInt(config.PyroscopeCPU, 10, 8)
	totalCPURequired := (minerCPUInt * int64(config.NumberOfMiners)) + (poetCPUInt * int64(config.NumberOfPoets)) + kibanaCPUInt + esCPUInt + pyroscopeCPUInt
	totalCPUInstanceHas := int64(config.GCPMachineCPU)

	nodeCount1 := int(math.Ceil((float64(totalCPURequired) / float64(totalCPUInstanceHas))))

	minerMemoryInt, _ := strconv.ParseInt(config.MinerMemory, 10, 8)
	poetMemoryInt, _ := strconv.ParseInt(config.PoetMemory, 10, 8)
	esMemoryInt, _ := strconv.ParseInt(config.ESMemory, 10, 8)
	kibanaMemoryInt, _ := strconv.ParseInt(config.KibanaMemory, 10, 8)
	pyroscopeMemoryInt, _ := strconv.ParseInt(config.PyroscopeMemory, 10, 8)
	totalMemoryRequired := (minerMemoryInt * int64(config.NumberOfMiners)) + (poetMemoryInt * int64(config.NumberOfPoets)) + kibanaMemoryInt + esMemoryInt + pyroscopeMemoryInt
	totalMemoryInstanceHas := int64(config.GCPMachineMemory)

	nodeCount2 := int(math.Ceil((float64(totalMemoryRequired) / float64(totalMemoryInstanceHas))))

	nodeCount := 0

	if nodeCount1 > nodeCount2 {
		nodeCount = nodeCount1
	} else {
		nodeCount = nodeCount2
	}

	if nodeCount == 0 {
		nodeCount = 1
	}

	accelerators := []*containerpb.AcceleratorConfig{}

	if config.AcceletatorType != "" {
		accelerators = append(accelerators, &containerpb.AcceleratorConfig{
			AcceleratorCount: config.AcceleratorCount,
			AcceleratorType:  config.AcceletatorType,
		})
	}

	nodePools := [](*containerpb.NodePool){
		&containerpb.NodePool{
			Name:             "default",
			InitialNodeCount: int32(nodeCount),
			Autoscaling: &containerpb.NodePoolAutoscaling{
				Enabled:      true,
				MaxNodeCount: 1000,
			},
			Config: &containerpb.NodeConfig{
				MachineType:  config.GCPMachineType,
				Accelerators: accelerators,
			},
			Locations: []string{config.GCPZone},
			Management: &containerpb.NodeManagement{
				AutoUpgrade: false,
				AutoRepair:  false,
			},
		},
	}

	cluster := &containerpb.Cluster{
		Name:                  config.NetworkName,
		NodePools:             nodePools,
		InitialClusterVersion: "1.21.12-gke.1700", //https://cloud.google.com/kubernetes-engine/docs/release-notes
		ReleaseChannel: &containerpb.ReleaseChannel{
			Channel: containerpb.ReleaseChannel_UNSPECIFIED,
		},
	}

	if config.UseVPC {
		cluster.Network = config.VPC
	}

	req := &containerpb.CreateClusterRequest{
		Cluster: cluster,
		Parent:  "projects/" + config.GCPProject + "/locations/" + config.GCPLocation,
	}

	fmt.Println("creating k8s cluster")

	_, err = client.CreateCluster(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Println("created k8s cluster")
	fmt.Println("waiting for k8s cluster to be ready")

	for range time.Tick(10 * time.Second) {
		cluster, err := getCluster(config.NetworkName)

		if err != nil {
			return err
		}

		fmt.Println("current k8s cluster status: ", cluster.Status)

		if cluster.Status == containerpb.Cluster_PROVISIONING || cluster.Status == containerpb.Cluster_STATUS_UNSPECIFIED || cluster.Status == containerpb.Cluster_RECONCILING {
			continue
		} else if cluster.Status == containerpb.Cluster_RUNNING {
			break
		} else if cluster.Status == containerpb.Cluster_STOPPING || cluster.Status == containerpb.Cluster_ERROR || cluster.Status == containerpb.Cluster_DEGRADED {
			return fmt.Errorf("an unknown occured while k8s cluster was being created. status: ", cluster.Status)
		}
	}

	fmt.Println("k8s cluster is ready")

	return nil
}

func GetKubernetesClient(networkName string) (*restclient.Config, *kubernetes.Clientset, error) {
	cluster, err := getCluster(networkName)

	if err != nil {
		return nil, nil, err
	}

	ret := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   map[string]*api.Cluster{},  // Clusters is a map of referencable names to cluster configs
		AuthInfos:  map[string]*api.AuthInfo{}, // AuthInfos is a map of referencable names to user configs
		Contexts:   map[string]*api.Context{},  // Contexts is a map of referencable names to context configs
	}
	name := fmt.Sprintf("gke_%s_%s_%s", config.GCPProject, cluster.Zone, cluster.Name)
	cert, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)

	if err != nil {
		return nil, nil, err
	}

	ret.Clusters[name] = &api.Cluster{
		CertificateAuthorityData: cert,
		Server:                   "https://" + cluster.Endpoint,
	}

	ret.Contexts[name] = &api.Context{
		Cluster:  name,
		AuthInfo: name,
	}

	ret.AuthInfos[name] = &api.AuthInfo{
		AuthProvider: &api.AuthProviderConfig{
			Name: "gcp",
			Config: map[string]string{
				"scopes": "https://www.googleapis.com/auth/cloud-platform",
			},
		},
	}

	cfg, err := clientcmd.NewNonInteractiveClientConfig(ret, name, &clientcmd.ConfigOverrides{CurrentContext: name}, nil).ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes configuration cluster=%s: %w", name, err)
	}

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Kubernetes client cluster=%s: %w", name, err)
	}

	return cfg, k8s, nil
}

func DeleteKubernetesCluster(volumes []string) error {

	ctx := context.Background()
	computeService, err := compute.NewService(ctx)

	if err != nil {
		return err
	}

	pageToken := ""

	volumesToDelete := []string{}

	for {
		disks := computeService.Disks.List(config.GCPProject, config.GCPZone)
		if pageToken != "" {
			disks = disks.PageToken(pageToken)
		} else {
			disks.MaxResults(100)
		}

		list, err := disks.Do()

		if err != nil {
			return err
		}

		for _, disk := range list.Items {
			str := strings.Split(disk.Name, "pvc")

			if len(str) == 2 {
				volumeName := "pvc" + str[1]

				if contains(volumes, volumeName) == true {
					volumesToDelete = append(volumesToDelete, disk.Name)
				}
			}
		}

		if list.NextPageToken != "" {
			pageToken = list.NextPageToken
		} else {
			break
		}
	}

	client, err := getClient()

	if err != nil {
		return err
	}

	req := &containerpb.DeleteClusterRequest{
		Name: "projects/" + config.GCPProject + "/locations/" + config.GCPLocation + "/clusters/" + config.NetworkName,
	}

	_, err = client.DeleteCluster(context.Background(), req)

	fmt.Println("started deleting cluster")

	if err != nil {
		return err
	}

	for range time.Tick(time.Duration(10) * time.Second) {
		_, err = getCluster(config.NetworkName)

		if err != nil {
			break
		}

		fmt.Println("waiting for cluster to delete")
	}

	fmt.Println("cluster deleted")

	for _, name := range volumesToDelete {
		fmt.Println("deleting disk: " + name)
		disk := computeService.Disks.Delete(config.GCPProject, config.GCPZone, name)
		_, err := disk.Do()

		if err != nil {
			return err
		}
	}

	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func ResizeKubernetesClusterForLogs() error {
	client, err := getClient()

	if err != nil {
		return err
	}

	_, err = client.SetNodePoolSize(context.TODO(), &containerpb.SetNodePoolSizeRequest{
		//autoscaler kicks in
		NodeCount: int32(1),
		Name:      "projects/" + config.GCPProject + "/locations/" + config.GCPLocation + "/clusters/" + config.NetworkName + "/nodePools/default",
	})

	if err != nil {
		return err
	}

	return nil
}
