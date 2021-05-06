package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rewardsCmd = &cobra.Command{
	Use:   "rewards",
	Short: "Rewards of the network",
	Run: func(cmd *cobra.Command, args []string) {
		err := network.Rewards()
		if err != nil {
			log.Error.Println(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(rewardsCmd)

	rewardsCmd.Flags().StringVar(&config.Host, "host", config.Host, "host to connect to")

	err := viper.BindPFlags(rewardsCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
