package network

import (
	"github.com/spacemeshos/spacecraft/gcp"
)

func Delete() error {
	err := gcp.DeleteKubernetesCluster()

	if err != nil {
		return err
	}

	return nil
}
