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

	"github.com/spacemeshos/spacecraft/log"
	"github.com/spacemeshos/spacecraft/network"
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
		}

		log.Success.Println("miner added successfully")
	},
}

func init() {
	rootCmd.AddCommand(addMinerCmd)

	addMinerCmd.Flags().StringVar(&config.MinerGoSmConfig, "miner-go-sm-config", config.MinerGoSmConfig, "config file location for the new miner (example \"./config.json\")")
	addMinerCmd.Flags().StringVar(&config.MinerNumber, "miner-number", config.MinerNumber, "miner to add")

	err := viper.BindPFlags(addMinerCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
