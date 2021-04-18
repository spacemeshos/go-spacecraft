package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var upgradeNetworkCmd = &cobra.Command{
	Use:   "upgradeNetwork",
	Short: "Upgrade a network",
	Long: `Used to upgrade go-sm build for all miners in a network. For example:

spacecraft upgradeNetwork --go-sm-image=spacemeshos/go-spacemesh:v0.1.26`,
	Run: func(cmd *cobra.Command, args []string) {
		err := network.Upgrade()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("miners upgraded successfully")
	},
}

func init() {
	rootCmd.AddCommand(upgradeNetworkCmd)

	upgradeNetworkCmd.Flags().StringVar(&config.GoSmImage, "go-sm-image", config.GoSmImage, "docker image for go-spacemesh build")
	upgradeNetworkCmd.Flags().IntVar(&config.RestartWaitTime, "restart-wait-time", config.RestartWaitTime, "sleep time between miner restarts in minutes")

	err := viper.BindPFlags(upgradeNetworkCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
