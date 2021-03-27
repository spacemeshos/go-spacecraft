package cmd

import (
	"fmt"
	"strings"

	cfg "github.com/spacemeshos/spacecraft/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var config = &cfg.Config

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
	rootCmd.PersistentFlags().StringVar(&config.GCPAuthFile, "gcp-auth-file", config.GCPAuthFile, "gcp json key file path")
	rootCmd.PersistentFlags().StringVar(&config.GCPLocation, "gcp-location", config.GCPLocation, "gcp cluster location")
	rootCmd.PersistentFlags().StringVar(&config.GCPProject, "gcp-project", config.GCPProject, "gcp project")

	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func initConfig() {
	viper.SetEnvPrefix("spacecraft")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		viper.ReadInConfig()
	}

	viper.Unmarshal(&config)
}
