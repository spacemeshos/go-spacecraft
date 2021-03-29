package gcp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	container "cloud.google.com/go/container/apiv1"
	cfg "github.com/spacemeshos/spacecraft/config"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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

	// c, err := container.NewClusterManagerClient(ctx, option.WithCredentialsFile(config.GCPAuthFile))
	c, err := container.NewClusterManagerClient(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not authorize gcp", err)
	}

	return c, nil
}

func getCluster() (*containerpb.Cluster, error) {
	client, err := getClient()

	if err != nil {
		return nil, err
	}

	req := &containerpb.GetClusterRequest{
		Name: "projects/" + config.GCPProject + "/locations/" + config.GCPLocation + "/clusters/" + config.NetworkName,
	}

	return client.GetCluster(context.Background(), req)
}

func CreateKubernetesCluster() error {
	client, err := getClient()

	if err != nil {
		return err
	}

	_, err = getCluster()

	if err == nil {
		return errors.New("cluster already exists")
	}

	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			return err
		}
	}

	// fmt.Printf("%+v", containerpb.Cluster{
	// 	Name:             config.NetworkName,
	// 	InitialNodeCount: 1,
	// })

	nodePools := [](*containerpb.NodePool){
		&containerpb.NodePool{
			Name:             "default",
			InitialNodeCount: 1,
			Autoscaling: &containerpb.NodePoolAutoscaling{
				Enabled:      true,
				MaxNodeCount: 1000,
			},
			Config: &containerpb.NodeConfig{
				MachineType: config.GCPMachineType,
			},
		},
	}

	req := &containerpb.CreateClusterRequest{
		Cluster: &containerpb.Cluster{
			Name:      config.NetworkName,
			NodePools: nodePools,
		},
		Parent: "projects/" + config.GCPProject + "/locations/" + config.GCPLocation,
	}

	fmt.Println("creating k8s cluster")

	_, err = client.CreateCluster(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Println("created k8s cluster")
	fmt.Println("waiting for k8s cluster to be ready")

	for range time.Tick(10 * time.Second) {
		cluster, err := getCluster()

		if err != nil {
			return err
		}

		fmt.Println("current k8s cluster status status: ", cluster.Status)

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

func GetKubernetesClient() (*kubernetes.Clientset, error) {
	cluster, err := getCluster()
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
		return nil, err
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
		return nil, fmt.Errorf("failed to create Kubernetes configuration cluster=%s: %w", name, err)
	}

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client cluster=%s: %w", name, err)
	}

	// ns, err := k8s.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to list namespaces cluster=%s: %w", name, err)
	// }

	// fmt.Printf("Namespaces found in cluster=%s", name)

	// for _, item := range ns.Items {
	// 	fmt.Println(item.Name)
	// }

	return k8s, nil
}
