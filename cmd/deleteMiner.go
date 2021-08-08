package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deleteMinerCmd = &cobra.Command{
	Use:   "deleteMiner",
	Short: "Delete a miner",
	Long: `Delete a individual miner from the network. For example:

spacecraft deleteMiner --miner=1`,
	Run: func(cmd *cobra.Command, args []string) {
		err := network.DeleteMiner()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("miner delete successfully")
	},
}

func init() {
	rootCmd.AddCommand(deleteMinerCmd)

	deleteMinerCmd.Flags().StringVar(&config.MinerNumber, "miner-number", config.MinerNumber, "miner to delete")
	deleteMinerCmd.Flags().StringVar(&config.SlackToken, "slack-token", config.SlackToken, "slack API token to post alerts")
	deleteMinerCmd.Flags().StringVar(&config.SlackChannelId, "slack-channel-id", config.SlackChannelId, "slack channel ID to post alerts")

	err := viper.BindPFlags(deleteMinerCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
