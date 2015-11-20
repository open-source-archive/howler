//handles the configuration of the applications. Yaml files are mapped with the struct

package conf

import (
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

type Config struct {
	BackendType       string //"marathon" or "kubernetes"
	Endpoint          string //URL of the backend
	FluentdEnabled    bool   //true if fluentd is enabled, will be ON for each container
	DebugEnabled      bool
	Oauth2Enabled     bool //true if authentication is enabled
	TeamAuthorization bool //true if authorization is enabled and based on teams
	AuthURL           string
	TokenURL          string
	TlsCertfilePath   string
	TlsKeyfilePath    string
	LogFlushInterval  time.Duration
	Port              int
	AuthorizedUsers   []AccessTuple
	Backends          map[string]map[string]string
	VersionBuildStamp string
	VersionGitHash    string
}

type AccessTuple struct {
	Realm string
	Uid   string
	Cn    string
}

//created a struct just for future usage
type ConfigError struct {
	Message string
}

//shared state for configuration
var conf *Config

//GetConfig gets the loaded configuration
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

func configInit(filename string) (*Config, *ConfigError) {
	viper := viper.New()
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/pmi-monitoring-connector")
	viper.AddConfigPath(fmt.Sprintf("%s/.config/pmi-monitoring-connector", os.ExpandEnv("$HOME")))
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
