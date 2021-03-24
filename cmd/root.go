package cmd

import (
	"fmt"

	cfg "github.com/spacemeshos/spacecraft/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var config = cfg.DefaultConfig()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "spacecraft",
	Short: "A CLI tool to create and manage spacemesh networks on GBP",
	Long:  `It supports creating network, adding/removing miners to an existing network, upgrading nodes in an network and replacing an existing network. It also deploys ELK for log analysis and prometheus/grafana for monitoring.`,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.PersistentFlags().StringVarP(&config.NetworkName, "network-name", "n", config.NetworkName, "name of the network")

	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
		viper.Unmarshal(&config)
	}
}
