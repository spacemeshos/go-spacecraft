/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
	addMinerCmd.Flags().BoolVar(&config.OldAPIExists, "old-api-exists", config.OldAPIExists, "does the go-spacemesh build support the old API")
	addMinerCmd.Flags().StringVar(&config.MinerMemory, "miner-ram", config.MinerMemory, "RAM for each miner")
	addMinerCmd.Flags().StringVar(&config.MinerCPU, "miner-cpu", config.MinerCPU, "vCPUs for each miner")
	addMinerCmd.Flags().StringVar(&config.GoSmImage, "go-sm-image", config.GoSmImage, "docker image for go-spacemesh build")
	addMinerCmd.Flags().StringVar(&config.MinerDiskSize, "miner-disk-size", config.MinerDiskSize, "Disk size of miner in GB")
	addMinerCmd.Flags().BoolVar(&config.Metrics, "metrics", config.Metrics, "enable go-sm metrics collection")
	addMinerCmd.Flags().StringVar(&config.SlackToken, "slack-token", config.SlackToken, "slack API token to post alerts")
	addMinerCmd.Flags().StringVar(&config.SlackChannelId, "slack-channel-id", config.SlackChannelId, "slack channel ID to post alerts")

	err := viper.BindPFlags(addMinerCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
