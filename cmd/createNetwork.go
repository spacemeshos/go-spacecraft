package cmd

import (
	"fmt"

	"github.com/spacemeshos/go-spacecraft/log"
	"github.com/spacemeshos/go-spacecraft/network"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var createNetworkCmd = &cobra.Command{
	Use:   "createNetwork",
	Short: "Create a network",
	Long: `Create a new network with N miners and N PoETs. For example:
	
spacecraft createNetwork -m=10 -p=3

Or else create a network that joins another network using a config file generated by parent network. For example:

spacecraft createNetwork -m=10 -p=3 --network-name=devnet1 --bootstrap=false --miner-go-sm-config=./config.json
`,
	Run: func(cmd *cobra.Command, args []string) {

		err := network.Create()
		if err != nil {
			log.Error.Println(err)
			return
		}

		log.Success.Println("network created successfully")
	},
}

func init() {
	rootCmd.AddCommand(createNetworkCmd)

	createNetworkCmd.Flags().StringVar(&config.GoSmConfig, "go-sm-config", config.GoSmConfig, "config file for go-spacemesh")
	createNetworkCmd.Flags().StringVar(&config.KibanaSavedObjects, "kibana-saved-objects", config.KibanaSavedObjects, "path of file exported using kibana export API")
	createNetworkCmd.Flags().StringVar(&config.ESCert, "es-cert", config.ESCert, "path to p12 file for elasticsearch ssl")
	createNetworkCmd.Flags().IntVarP(&config.NumberOfMiners, "miners", "m", config.NumberOfMiners, "number of miners")
	createNetworkCmd.Flags().IntVarP(&config.NumberOfPoets, "poets", "p", config.NumberOfPoets, "number of poets")
	createNetworkCmd.Flags().StringVar(&config.MinerMemory, "miner-ram", config.MinerMemory, "RAM for each miner")
	createNetworkCmd.Flags().StringVar(&config.MinerCPU, "miner-cpu", config.MinerCPU, "vCPUs for each miner")
	createNetworkCmd.Flags().StringVar(&config.PoetMemory, "poet-ram", config.PoetMemory, "RAM for each poet")
	createNetworkCmd.Flags().StringVar(&config.PoetCPU, "poet-cpu", config.PoetCPU, "vCPUs for each poet")
	createNetworkCmd.Flags().StringVar(&config.MinerDiskSize, "miner-disk-size", config.MinerDiskSize, "Disk size of miner in GB")
	createNetworkCmd.Flags().StringVar(&config.PoetDiskSize, "poet-disk-size", config.PoetDiskSize, "Disk size of poet in GB")
	createNetworkCmd.Flags().StringVar(&config.GoSmImage, "go-sm-image", config.GoSmImage, "docker image for go-spacemesh build")
	createNetworkCmd.Flags().StringVar(&config.PoetImage, "poet-image", config.PoetImage, "docker image for poet build")
	createNetworkCmd.Flags().IntVar(&config.PoetGatewayAmount, "poet-gateway-amount", config.PoetGatewayAmount, "number of gateway to pass when activating poet(s)")
	createNetworkCmd.Flags().IntVar(&config.BootnodeAmount, "bootnode-amount", config.BootnodeAmount, "total bootnodes in the generated config file")
	createNetworkCmd.Flags().IntVar(&config.GCPMachineCPU, "gcp-machine-cpu", config.GCPMachineCPU, "total CPU the GCP machine type has")
	createNetworkCmd.Flags().IntVar(&config.GCPMachineMemory, "gcp-machine-memory", config.GCPMachineMemory, "total memory the GCP machine type has")
	createNetworkCmd.Flags().IntVar(&config.GenesisDelay, "genesis-delay", config.GenesisDelay, "delay in minutes after network startup for genesis")
	createNetworkCmd.Flags().BoolVar(&config.Bootstrap, "bootstrap", config.Bootstrap, "bootstrap a new network without connecting to an existing network")
	createNetworkCmd.Flags().StringVar(&config.MinerGoSmConfig, "miner-go-sm-config", config.MinerGoSmConfig, "config file location for the miners (example \"./config.json\")")
	createNetworkCmd.Flags().StringVar(&config.ESDiskSize, "es-disk-size", config.ESDiskSize, "disk size to allocate to elasticsearch")
	createNetworkCmd.Flags().StringVar(&config.ESCPU, "es-cpu", config.ESCPU, "vCPUs to allocate to elasticsearch")
	createNetworkCmd.Flags().StringVar(&config.ESMemory, "es-memory", config.ESMemory, "RAM to allocate to elasticsearch")
	createNetworkCmd.Flags().StringVar(&config.ESHeapMemory, "es-heap-memory", config.ESHeapMemory, "memory to allocate to elasticsearch heap (should be less than total memory allocated")
	createNetworkCmd.Flags().StringVar(&config.ESReplicas, "es-replicas", config.ESReplicas, "number of ES nodes")
	createNetworkCmd.Flags().StringVar(&config.ESMasterNodes, "es-master-nodes", config.ESMasterNodes, "number of master nodes out of total nodes")
	createNetworkCmd.Flags().StringVar(&config.KibanaCPU, "kibana-cpu", config.KibanaCPU, "vCPUs to allocate to kibana")
	createNetworkCmd.Flags().StringVar(&config.KibanaMemory, "kibana-memory", config.KibanaMemory, "RAM to allocate to kibana")
	createNetworkCmd.Flags().StringVar(&config.LogsExpiry, "logs-expiry", config.LogsExpiry, "number of days after which logs are deleted automatically")
	createNetworkCmd.Flags().BoolVar(&config.OldAPIExists, "old-api-exists", config.OldAPIExists, "does the go-spacemesh build support the old API")
	createNetworkCmd.Flags().BoolVar(&config.AdjustHare, "adjust-hare", config.AdjustHare, "adjust hare parameters according to number of miners")
	createNetworkCmd.Flags().StringVar(&config.PyroscopeImage, "pyroscope-image", config.PyroscopeImage, "docker image url of pyroscope")
	createNetworkCmd.Flags().StringVar(&config.PyroscopeCPU, "pyroscope-cpu", config.PyroscopeCPU, "vCPUs to allocate to pyroscope")
	createNetworkCmd.Flags().StringVar(&config.PyroscopeMemory, "pyroscope-memory", config.PyroscopeMemory, "memory to allocate to pyroscope")
	createNetworkCmd.Flags().BoolVar(&config.DeployPyroscope, "deploy-pyroscope", config.DeployPyroscope, "deploy pyroscope profiler")
	createNetworkCmd.Flags().BoolVar(&config.Metrics, "metrics", config.Metrics, "enable go-sm metrics collection")
	createNetworkCmd.Flags().BoolVar(&config.EnableJsonAPI, "enable-json-api", config.EnableJsonAPI, "enables JSON api in all nodes")
	createNetworkCmd.Flags().IntVar(&config.MaxConcurrentDeployments, "max-concurrent-deployments", config.MaxConcurrentDeployments, "number of miners that can be deployed concurrently")
	createNetworkCmd.Flags().BoolVar(&config.EnableGoDebug, "enable-go-debug", config.EnableGoDebug, "start miners with GODEBUG=\"gctrace=1,scavtrace=1,gcpacertrace=1\" env")
	createNetworkCmd.Flags().StringVar(&config.PrometheusMemory, "prometheus-memory", config.PrometheusMemory, "memory to allocate to prometheus in GB")
	createNetworkCmd.Flags().StringVar(&config.PrometheusCPU, "prometheus-cpu", config.PrometheusCPU, "vCPU to allocate to prometheus")
	createNetworkCmd.Flags().StringVar(&config.AcceletatorType, "accelerator-type", config.AcceletatorType, "VM GPU accelerator type (https://cloud.google.com/compute/docs/gpus)")
	createNetworkCmd.Flags().Int64Var(&config.AcceleratorCount, "accelerator-count", config.AcceleratorCount, "number of accelerators")
	createNetworkCmd.Flags().StringVar(&config.GCPMachineType, "gcp-machine-type", config.GCPMachineType, "VM machine type")
	createNetworkCmd.Flags().StringVar(&config.SlackToken, "slack-token", config.SlackToken, "slack API token to post alerts")
	createNetworkCmd.Flags().StringVar(&config.SlackChannelId, "slack-channel-id", config.SlackChannelId, "slack channel ID to post alerts")
	createNetworkCmd.Flags().BoolVar(&config.EnableSlackAlerts, "enable-slack-alerts", config.EnableSlackAlerts, "deploy spacemesh-watch")

	err := viper.BindPFlags(createNetworkCmd.Flags())
	if err != nil {
		fmt.Println("an error has occurred while binding flags:", err)
	}
}
