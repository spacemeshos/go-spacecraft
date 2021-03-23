package cmd

import (
	"github.com/spacemeshos/spacecraft/config"
	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "spacecraft",
	Short: "A CLI tool to create and manage spacemesh networks on GBP",
	Long:  `It supports creating network, adding/removing miners to an existing network, upgrading nodes in an network and replacing an existing network. It also deploys ELK for log analysis and prometheus/grafana for monitoring.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&config.NetworkName, "network-name", "n", "", "name of the network")
	rootCmd.MarkPersistentFlagRequired("network-name")
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
