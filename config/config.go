package config

type Configuration struct {
	NetworkName    string `mapstructure:"network-name"`
	NumberOfMiners int    `mapstructure:"miners"`
	NumberOfPoets  int    `mapstructure:"poets"`
	MinerMemory    int    `mapstructure:"miner-ram"`
	MinerCPU       int    `mapstructure:"miner-cpu"`
	PoetMemory     int    `mapstructure:"poet-ram"`
	PoetCPU        int    `mapstructure:"poet-cpu"`
	MinerDiskSize  int    `mapstructure:"miner-disk-size"`
	PoetDiskSize   int    `mapstructure:"poet-disk-size"`
	GoSmImage      string `mapstructure:"go-sm-image"`
	PoetImage      string `mapstructure:"poet-image"`
}

var Config = Configuration{
	NetworkName:    "devnet",
	NumberOfMiners: 10,
	NumberOfPoets:  3,
	MinerMemory:    4,
	MinerCPU:       2,
	PoetMemory:     4,
	PoetCPU:        2,
	MinerDiskSize:  50,
	PoetDiskSize:   50,
	GoSmImage:      "spacemeshos/go-spacemesh:v0.1.24",
	PoetImage:      "spacemeshos/poet:73488d6",
}
