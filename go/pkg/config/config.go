package config

// Config holds global configuration values
type Config struct {
	Verbose    bool
	ConfigFile string // Path to brewDeps.yaml
}

// Global configuration instance
var Global = Config{
	ConfigFile: "brewDeps.yaml", // Default value
}
