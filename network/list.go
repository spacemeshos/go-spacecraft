package network

import (
	"github.com/spacemeshos/go-spacecraft/gcp"
	"github.com/spacemeshos/go-spacecraft/log"
)

func ListNetworks() error {
	networks, err := gcp.GetClusters()

	if err != nil {
		return err
	}

	if len(networks) == 0 {
		log.Error.Println("No networks found")
	} else {
		log.Info.Println("Here is the list of deployed networks:")
		for _, name := range networks {
			log.Success.Println(name)
		}
	}

	return nil
}
