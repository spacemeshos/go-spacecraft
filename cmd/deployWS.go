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

	deployWSCmd.Flags().StringVar(&config.MinerMemory, "miner-ram", config.MinerMemory, "RAM for each miner")
	deployWSCmd.Flags().StringVar(&config.MinerCPU, "miner-cpu", config.MinerCPU, "vCPUs for each miner")
	deployWSCmd.Flags().StringVar(&config.GoSmImage, "go-sm-image", config.GoSmImage, "docker image for go-spacemesh build")
	deployWSCmd.Flags().StringVar(&config.DashboardVersion, "dash-version", config.DashboardVersion, "docker image tag for spacemeshos/dash-backend")
	deployWSCmd.Flags().StringVar(&config.ExplorerVersion, "explorer-version", config.ExplorerVersion, "docker image tag for spacemeshos/explorer-apiserver and spacemeshos/explorer-collector")

	err := viper.BindPFlags(deployWSCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
