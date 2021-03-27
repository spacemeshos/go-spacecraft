package gcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	container "cloud.google.com/go/container/apiv1"
	cfg "github.com/spacemeshos/spacecraft/config"
	"github.com/spacemeshos/spacecraft/log"
	"google.golang.org/api/option"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

var config = &cfg.Config

func getClient() *container.ClusterManagerClient {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if config.GCPAuthFile == "" || config.GCPProject == "" || config.GCPLocation == "" {
		log.Error.Println("please provide JSON key file, project name and location for gcp authorization")
		os.Exit(126)
	}

	c, err := container.NewClusterManagerClient(ctx, option.WithCredentialsFile(config.GCPAuthFile))

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
}
