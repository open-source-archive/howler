//handles the configuration of the applications. Yaml files are mapped with the struct

package conf

import (
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

// Config provides the base fields to start Howler
type Config struct {
	DebugEnabled     bool
	Oauth2Enabled    bool //true if authentication is enabled
	AuthURL          string
	TokenURL         string
	TLSCertfilePath  string
	TLSKeyfilePath   string
	LogFlushInterval time.Duration
	Port             int
	AuthorizedUsers  []AccessTuple
	Backends         map[string]map[string]string
	PrintVersion     bool
	Version          string
	BuildStamp       string
	GitHash          string
}

// AccessTuple provides fields verifying users
type AccessTuple struct {
	Realm string
	UID   string
	Cn    string
}

//ConfigError creates a struct just for future usage
type ConfigError struct {
	Message string
}

//conf shares state for configuration
var conf *Config

//New gets the loaded configuration
func New() *Config {
	var err *ConfigError
	if conf == nil {
		conf, err = configInit("config.yaml")
		if err != nil {
			glog.Errorf("could not load configuration. Reason: %s", err.Message)
			panic("Cannot load configuration. Exiting.")
		}
	}
	return conf
}

//configInit initializes Howler configuration
func configInit(filename string) (*Config, *ConfigError) {
	viper := viper.New()
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/howler")
	viper.AddConfigPath(fmt.Sprintf("%s/.config/howler", os.ExpandEnv("$HOME")))
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Can not read config, caused by: %s\n", err)
		return nil, &ConfigError{"cannot read configuration, something must be wrong."}
	}
	var config Config
	err = viper.Marshal(&config)
	if err != nil {
		fmt.Printf("Can not marshal config, caused by: %s\n", err)
		return nil, &ConfigError{"configuration format is not correct."}
	}
	return &config, nil
}
