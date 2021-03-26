package network

import (
	"context"
	"os"
	"time"

	container "cloud.google.com/go/container/apiv1"
	cfg "github.com/spacemeshos/spacecraft/config"
	"github.com/spacemeshos/spacecraft/log"
	"google.golang.org/api/option"
)

var config = &cfg.Config

func GetClient() (client *container.ClusterManagerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if config.GCPAuthFile == "" {
		log.Error.Println("please provide JSON key file for gcp authorization")
		os.Exit(126)
	}

	client, err := container.NewClusterManagerClient(ctx, option.WithCredentialsFile(config.GCPAuthFile))

	if err != nil {
		log.Error.Println("could not authorize gcp", err)
		os.Exit(126)
	}

	return
}
