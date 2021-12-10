package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var releaseNetworkCmd = &cobra.Command{
	Use:   "releaseNetwork",
	Short: "Release a network to sm-net repo",
	Long:  `For example: spacecraft releaseNetwork`,
	Run: func(cmd *cobra.Command, args []string) {
		err := network.ReleaseNetwork()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("network released successfully")
	},
}

func init() {
	rootCmd.AddCommand(releaseNetworkCmd)

	releaseNetworkCmd.Flags().StringVar(&config.GoSmReleaseVersion, "go-sm-release-version", config.GoSmReleaseVersion, "go-spacemesh tag/release version")
	releaseNetworkCmd.Flags().StringVar(&config.GithubToken, "github-token", config.GithubToken, "github token for authentication")

	err := viper.BindPFlags(releaseNetworkCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
