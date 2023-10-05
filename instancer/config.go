package instancer

import (
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type Config struct {
	InstanceTTL string
	ListenAddr  string
	LogLevel    zerolog.Level
	LogRequests bool
	DBFile      string
	APIToken    string
}

const DEFAULT_CONFIG_FILE = "instanced.yaml"

func loadConfig(log zerolog.Logger) Config {
	// Initalize Settings
	v := viper.New()
	v.SetConfigFile(DEFAULT_CONFIG_FILE)
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/instanced/")
	v.AddConfigPath(".")
	v.SetEnvPrefix("insd")
	v.SetEnvKeyReplacer(strings.NewReplacer("_", "-"))
	v.AutomaticEnv()

	// Set defaults
	// How long each instance lasts by default
	v.SetDefault("instance-expiry", "10m")
	// Listen address for API server ip:port
	v.SetDefault("listen-addr", ":8080")
	// Zerolog log level string
	v.SetDefault("log-level", "info")
	// Log API requests
	v.SetDefault("log-request", true)
	// Sqlite DB file path
	v.SetDefault("db-file", "/data/instancer.db")
	// API Auth Token
	v.SetDefault("api-token", "token")

	// Read Config from file
	err := v.ReadInConfig()
	if err != nil {
		log.Error().Err(err).Msg("error reading config file")
	}

	// Populate config struct
	conf := Config{}
	conf.LogLevel, err = zerolog.ParseLevel(v.GetString("log-level"))
	if err != nil {
		log.Warn().Err(err).Msg("error parsing config")
	}
	conf.InstanceTTL = v.GetString("instance-expiry")
	conf.ListenAddr = v.GetString("listen-addr")
	conf.LogRequests = v.GetBool("log-request")
	conf.DBFile = v.GetString("db-file")
	conf.APIToken = v.GetString("api-token")
	return conf
}
