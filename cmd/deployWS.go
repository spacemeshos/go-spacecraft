package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployWSCmd = &cobra.Command{
	Use:   "deployWS",
	Short: "Deploys web services",
	Long:  `For example: spacecraft deployWS`,
	Run: func(cmd *cobra.Command, args []string) {
		err := network.DeployWS()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("web services deployed successfully")
	},
}

func init() {
	rootCmd.AddCommand(deployWSCmd)

	deployWSCmd.Flags().StringVar(&config.MinerGoSmConfig, "miner-go-sm-config", config.MinerGoSmConfig, "config file location (example \"./config.json\")")
	deployWSCmd.Flags().StringVar(&config.PeersFile, "miner-go-sm-peers", config.PeersFile, "peers.json file location (example \"./peers.json\")")
	deployWSCmd.Flags().StringVar(&config.GoSmImage, "go-sm-image", config.GoSmImage, "docker image for go-spacemesh build")

	err := viper.BindPFlags(deployWSCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
