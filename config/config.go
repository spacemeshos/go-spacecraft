package config

type Configuration struct {
	NetworkName              string `mapstructure:"network-name"`
	NumberOfMiners           int    `mapstructure:"miners"`
	NumberOfPoets            int    `mapstructure:"poets"`
	MinerMemory              string `mapstructure:"miner-ram"`
	MinerCPU                 string `mapstructure:"miner-cpu"`
	PoetMemory               string `mapstructure:"poet-ram"`
	PoetCPU                  string `mapstructure:"poet-cpu"`
	MinerDiskSize            string `mapstructure:"miner-disk-size"`
	PoetDiskSize             string `mapstructure:"poet-disk-size"`
	GoSmImage                string `mapstructure:"go-sm-image"`
	PoetImage                string `mapstructure:"poet-image"`
	GCPProject               string `mapstructure:"gcp-project"`
	GCPLocation              string `mapstructure:"gcp-location"`
	GCPZone                  string `mapstructure:"gcp-zone"`
	GCPMachineType           string `mapstructure:"gcp-machine-type"`
	GoSmConfig               string `mapstructure:"go-sm-config"`
	InitPhaseShift           int    `mapstructure:"init-phase-shift"`
	PoetGatewayAmount        int    `mapstructure:"poet-gateway-amount"`
	BootnodeAmount           int    `mapstructure:"bootnode-amount"`
	GCPMachineCPU            int    `mapstructure:"gcp-machine-cpu"`
	GCPMachineMemory         int    `mapstructure:"gcp-machine-memory"`
	GenesisDelay             int    `mapstructure:"genesis-delay"`
	MinerNumber              string `mapstructure:"miner-number"`
	MinerGoSmConfig          string `mapstructure:"miner-go-sm-config"`
	RestartWaitTime          int    `mapstructure:"restart-wait-time"`
	Bootstrap                bool   `mapstructure:"bootstrap"`
	KibanaSavedObjects       string `mapstructure:"kibana-saved-objects"`
	ESCert                   string `mapstructure:"es-cert"`
	ESDiskSize               string `mapstructure:"es-disk-size"`
	ESMemory                 string `mapstructure:"es-memory"`
	ESCPU                    string `mapstructure:"es-cpu"`
	ESHeapMemory             string `mapstructure:"es-heap-memory"`
	ESReplicas               string `mapstructure:"es-replicas"`
	ESMasterNodes            string `mapstructure:"es-master-nodes"`
	KibanaMemory             string `mapstructure:"kibana-memory"`
	KibanaCPU                string `mapstructure:"kibana-cpu"`
	LogsExpiry               string `mapstructure:"logs-expiry"`
	OldAPIExists             bool   `mapstructure:"old-api-exists"`
	AdjustHare               bool   `mapstructure:"adjust-hare"`
	Host                     string `mapstructure:"host"`
	PyroscopeImage           string `mapstructure:"pyroscope-image"`
	PyroscopeCPU             string `mapstructure:"pyroscope-cpu"`
	PyroscopeMemory          string `mapstructure:"pyroscope-memory"`
	DeployPyroscope          bool   `mapstructure:"deploy-pyroscope"`
	Metrics                  bool   `mapstructure:"metrics"`
	MaxConcurrentDeployments int    `mapstructure:"max-concurrent-deployments"`
	EnableJsonAPI            bool   `mapstructure:"enable-json-api"`
	EnableGoDebug            bool   `mapstructure:"enable-go-debug"`
	PrometheusMemory         string `mapstructure:"prometheus-memory"`
	PrometheusCPU            string `mapstructure:"prometheus-cpu"`
	AcceleratorCount         int64  `mapstructure:"accelerator-count"`
	AcceletatorType          string `mapstructure:"accelerator-type"`
	ImageType                string `mapstructure:"image-type"`
}

var Config = Configuration{
	NetworkName:              "mininet",
	NumberOfMiners:           10,
	NumberOfPoets:            1,
	MinerMemory:              "2",
	MinerCPU:                 "1",
	PoetMemory:               "2",
	PoetCPU:                  "1",
	MinerDiskSize:            "10", //200
	PoetDiskSize:             "10", //50
	GoSmImage:                "spacemeshos/go-spacemesh:v0.1.41",
	PoetImage:                "spacemeshos/poet:73488d6",
	GCPProject:               "",
	GCPLocation:              "",
	GCPZone:                  "",
	GCPMachineType:           "e2-standard-16",
	GCPMachineCPU:            16,
	GCPMachineMemory:         64,
	GoSmConfig:               "./artifacts/mininet/miner/config.json",
	InitPhaseShift:           0,
	PoetGatewayAmount:        4,
	BootnodeAmount:           6,
	GenesisDelay:             10,
	MinerNumber:              "",
	MinerGoSmConfig:          "",
	RestartWaitTime:          2,
	Bootstrap:                true,
	KibanaSavedObjects:       "./artifacts/elk/kibana.ndjson",
	ESCert:                   "./artifacts/elk/elastic-certificates.p12",
	ESDiskSize:               "10",
	ESMemory:                 "2",
	ESHeapMemory:             "1",
	ESCPU:                    "1",
	ESReplicas:               "1",
	ESMasterNodes:            "1",
	KibanaMemory:             "2",
	KibanaCPU:                "1",
	LogsExpiry:               "1",
	OldAPIExists:             true,
	AdjustHare:               true,
	Host:                     "",
	PyroscopeImage:           "pyroscope/pyroscope:latest",
	PyroscopeCPU:             "1",
	PyroscopeMemory:          "2",
	DeployPyroscope:          false,
	Metrics:                  false,
	MaxConcurrentDeployments: 100,
	EnableJsonAPI:            true,
	EnableGoDebug:            false,
	PrometheusMemory:         "1",
	PrometheusCPU:            "1",
	AcceleratorCount:         0,
	AcceletatorType:          "",
	ImageType:                "ubuntu-2010-groovy-v20210622a",
}
