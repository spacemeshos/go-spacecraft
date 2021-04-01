package config

type Configuration struct {
	NetworkName       string `mapstructure:"network-name"`
	NumberOfMiners    int    `mapstructure:"miners"`
	NumberOfPoets     int    `mapstructure:"poets"`
	MinerMemory       string `mapstructure:"miner-ram"`
	MinerCPU          string `mapstructure:"miner-cpu"`
	PoetMemory        string `mapstructure:"poet-ram"`
	PoetCPU           string `mapstructure:"poet-cpu"`
	MinerDiskSize     string `mapstructure:"miner-disk-size"`
	PoetDiskSize      string `mapstructure:"poet-disk-size"`
	GoSmImage         string `mapstructure:"go-sm-image"`
	PoetImage         string `mapstructure:"poet-image"`
	GCPProject        string `mapstructure:"gcp-project"`
	GCPLocation       string `mapstructure:"gcp-location"`
	GCPMachineType    string `mapstructure:"gcp-machine-type"`
	GoSmConfig        string `mapstructure:"go-sm-config"`
	InitPhaseShift    int    `mapstructure:"init-phase-shift"`
	PoetGatewayAmount int    `mapstructure:"poet-gateway-amount"`
	BootnodeAmount    int    `mapstructure:"bootnode-amount"`
	GCPMachineMemory  int    `mapstructure:"gcp-machine-memory"`
}

var Config = Configuration{
	NetworkName:       "devnet",
	NumberOfMiners:    50,
	NumberOfPoets:     1,
	MinerMemory:       "4",
	MinerCPU:          "2",
	PoetMemory:        "4",
	PoetCPU:           "2",
	MinerDiskSize:     "200",
	PoetDiskSize:      "50",
	GoSmImage:         "spacemeshos/go-spacemesh:v0.1.26",
	PoetImage:         "spacemeshos/poet:73488d6",
	GCPProject:        "",
	GCPLocation:       "",
	GCPMachineType:    "e2-standard-16",
	GCPMachineMemory:  16,
	GoSmConfig:        "./artifacts/devnet/miner/config.json",
	InitPhaseShift:    0,
	PoetGatewayAmount: 4,
	BootnodeAmount:    6,
}
