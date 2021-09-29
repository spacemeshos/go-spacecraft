package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addMinerCmd represents the addMiner command
var addMinerCmd = &cobra.Command{
	Use:   "addMiner",
	Short: "Add a miner to an existing network",
	Long:  `For example: spacecraft addMiner`,
	Run: func(cmd *cobra.Command, args []string) {
		err := network.AddMiner()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("miner added successfully")
	},
}

func init() {
	rootCmd.AddCommand(addMinerCmd)

	addMinerCmd.Flags().StringVar(&config.MinerGoSmConfig, "miner-go-sm-config", config.MinerGoSmConfig, "config file location for the new miner (example \"./config.json\")")
	addMinerCmd.Flags().StringVar(&config.MinerNumber, "miner-number", config.MinerNumber, "miner to add")
	addMinerCmd.Flags().StringVar(&config.MinerMemory, "miner-ram", config.MinerMemory, "RAM for each miner")
	addMinerCmd.Flags().StringVar(&config.MinerCPU, "miner-cpu", config.MinerCPU, "vCPUs for each miner")
	addMinerCmd.Flags().StringVar(&config.GoSmImage, "go-sm-image", config.GoSmImage, "docker image for go-spacemesh build")
	addMinerCmd.Flags().StringVar(&config.MinerDiskSize, "miner-disk-size", config.MinerDiskSize, "Disk size of miner in GB")
	addMinerCmd.Flags().BoolVar(&config.Metrics, "metrics", config.Metrics, "enable go-sm metrics collection")
	addMinerCmd.Flags().StringVar(&config.SlackToken, "slack-token", config.SlackToken, "slack API token to post alerts")
	addMinerCmd.Flags().StringVar(&config.SlackChannelId, "slack-channel-id", config.SlackChannelId, "slack channel ID to post alerts")
	addMinerCmd.Flags().BoolVar(&config.EnableSlackAlerts, "enable-slack-alerts", config.EnableSlackAlerts, "deploy spacemesh-watch")

	err := viper.BindPFlags(addMinerCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
