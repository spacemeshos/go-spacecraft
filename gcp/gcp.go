package gcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	container "cloud.google.com/go/container/apiv1"
	cfg "github.com/spacemeshos/spacecraft/config"
	"github.com/spacemeshos/spacecraft/log"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // register GCP auth provider
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var config = &cfg.Config

func getClient() *container.ClusterManagerClient {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if config.GCPProject == "" || config.GCPLocation == "" {
		log.Error.Println("please provide JSON key file, project name and location for gcp authorization")
		os.Exit(126)
	}

	// c, err := container.NewClusterManagerClient(ctx, option.WithCredentialsFile(config.GCPAuthFile))
	c, err := container.NewClusterManagerClient(ctx)

	if err != nil {
		log.Error.Println("could not authorize gcp", err)
		os.Exit(126)
	}

	return c
}

func getCluster() (*containerpb.Cluster, error) {
	client := getClient()

	req := &containerpb.GetClusterRequest{
		Name: "projects/" + config.GCPProject + "/locations/" + config.GCPLocation + "/clusters/" + config.NetworkName,
	}

	return client.GetCluster(context.Background(), req)
}

func CreateK8SCluster() {
	client := getClient()
	_, err := getCluster()

	if err == nil {
		log.Error.Println("cluster already exists")
		os.Exit(126)
	}

	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			log.Error.Println(err)
			os.Exit(126)
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
		log.Error.Println(err)
		os.Exit(126)
	}

	fmt.Println("created k8s cluster")
	fmt.Println("waiting for k8s cluster to be ready")

	for range time.Tick(10 * time.Second) {
		cluster, err := getCluster()

		if err != nil {
			log.Error.Println(err)
			os.Exit(126)
		}

		fmt.Println("current k8s cluster status status: ", cluster.Status)

		if cluster.Status == containerpb.Cluster_PROVISIONING || cluster.Status == containerpb.Cluster_STATUS_UNSPECIFIED || cluster.Status == containerpb.Cluster_RECONCILING {
			continue
		} else if cluster.Status == containerpb.Cluster_RUNNING {
			break
		} else if cluster.Status == containerpb.Cluster_STOPPING || cluster.Status == containerpb.Cluster_ERROR || cluster.Status == containerpb.Cluster_DEGRADED {
			log.Error.Println("an unknown occured while k8s cluster was being created. status: ", cluster.Status)
			os.Exit(126)
		}
	}

	fmt.Println("k8s cluster is ready")

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
		fmt.Println("error during 111")
		fmt.Println(err)
		os.Exit(126)
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
		fmt.Println("failed to create Kubernetes configuration cluster=%s: %w", name, err)
		os.Exit(126)
	}

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Println("failed to create Kubernetes client cluster=%s: %w", name, err)
		os.Exit(126)
	}

	ns, err := k8s.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("failed to list namespaces cluster=%s: %w", name, err)
		os.Exit(126)
	}

	fmt.Printf("Namespaces found in cluster=%s", name)

	for _, item := range ns.Items {
		fmt.Println(item.Name)
	}
}
