package config

type Config struct {
	NodeID            string   `yaml:"node_id"`
	Host              string   `yaml:"host"`
	RPCPort           int      `yaml:"rpc_port"`
	HTTPPort          int      `yaml:"http_port"`
	Peers             []string `yaml:"peers"`
	ElectionTimeout   int      `yaml:"election_timeout"` // in milliseconds
	HeartbeatInterval int      `yaml:"heartbeat_interval"` // in milliseconds
	DataDirectory     string   `yaml:"data_directory"`
}

func LoadConfig() *Config {
	// TODO: load config from file or env
	return &Config{}
}
