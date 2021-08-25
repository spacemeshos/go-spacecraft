package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deleteNetworkCmd = &cobra.Command{
	Use:   "deleteNetwork",
	Short: "Delete a network",
	Run: func(cmd *cobra.Command, args []string) {
		err := network.Delete()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("network deleted successfully")
	},
}

func init() {
	rootCmd.AddCommand(deleteNetworkCmd)

	deleteNetworkCmd.Flags().StringVar(&config.CloudflareAPIToken, "cloudflare-api-token", config.CloudflareAPIToken, "cloudflare API token")

	err := viper.BindPFlags(deleteNetworkCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
