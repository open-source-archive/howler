package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"gopkg.in/jmcvetta/napping.v3"
)

// @TODO: entity.ApplicationID should be postfixed with team name which owns the service, for now it's just "[techmonkeys]"

var usage = fmt.Sprintf(`
Usage:
    %s [options]

Options:
    --custom    this is my custom flag

Common Options:
  -h, --help            show this help message and exit
  -v, --verbose         Increase verbosity level (show debug messages)
  -q, --quiet           Decrease verbosity level
  -o, --onlydelete      Just delelte all matching ZMON entries
  -c CONFIG, --config CONFIG
                        Path to config file
  -l LOG_FILE, --log-file LOG_FILE
                        Path to log file
  -L LOCK_FILE, --lock-file LOCK_FILE
                        Path to lock file
  -s SLEEP, --sleep SLEEP
                        Sleep a random time before running

`, os.Args[0])

var log, _ = logging.GetLogger(os.Args[0])

const (
	ZMON_HOST     = "https://zmon2.zalando.net"
	ZMON_URL      = "/rest/api/v1/entities/"
	CONSUL_MASTER = "gth-consul01.zalando"
)

type Node struct {
	Node           string
	Address        string
	ServiceID      string
	ServiceName    string
	ServiceTags    []string
	ServiceAddress string
	ServicePort    int
}

type ZmonEntity struct {
	Type           string         `json:"type"`
	Id             string         `json:"id"`
	ApplicationID  string         `json:"application_id"`
	Host           string         `json:"host"`
	Ports          map[string]int `json:"ports"`
	DataCenterCode string         `json:"data_center_code"`
}

func maybeAbort(err error, msg string) {
	if err != nil {
		log.Fatalf("ERROR: %s %+v", msg, err)
	}
}

func readConfig(cf interface{}) {
	configFile, ok := cf.(string)

	if ok {
		viper.SetConfigFile(configFile)
	} else {
		log.Debug("looking for config file '%s.yaml' in /etc/, ~/.config/", os.Args[0])
		viper.SetConfigType("yaml")
		viper.SetConfigName(os.Args[0])
		viper.AddConfigPath("/etc/")
		viper.AddConfigPath(fmt.Sprintf("%s/.config/", os.ExpandEnv("$HOME")))
	}
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("unable to read config file '%s': %s", configFile, err)
	}
}

func notImplemented(option string) {
	fmt.Printf("Option %s is not implemented yet.\n", option)
}

func main() {
	var err error
	var response *napping.Response

	arguments, err := docopt.Parse(usage, nil, true, fmt.Sprintf("%s 0.1-dev", os.Args[0]), false)
	if err != nil {
		panic("Could not parse CLI")
	}

	if arguments["--log-file"] != nil {
		notImplemented("--log-file")
	}
	if arguments["--lock-file"] != nil {
		notImplemented("--lock-file")
	}
	if arguments["--sleep"] != nil {
		notImplemented("--sleep")
	}

	logging.SetLevel(logging.INFO, os.Args[0])
	if arguments["--verbose"].(bool) {
		logging.SetLevel(logging.DEBUG, os.Args[0])
	}
	if arguments["--quiet"].(bool) {
		logging.SetLevel(logging.WARNING, os.Args[0])
	}

	readConfig(arguments["--config"])

	zmonEntitiesServiceURL := ZMON_HOST + ZMON_URL
	consulBaseURL := fmt.Sprintf("https://%s:8500/v1/catalog", CONSUL_MASTER)
	datacenters := [...]string{"gth", "itr"}

	s := napping.Session{}
	s.Userinfo = url.UserPassword(viper.GetString("user"), viper.GetString("password"))
	s.Header = &http.Header{"Content-Type": []string{"application/json"}}

	// get all existing entities from ZMON
	query := map[string]string{"type": "service"}
	queryString, _ := json.Marshal(query)

	existingEntitiesURL := fmt.Sprintf("%s/?query=%s", zmonEntitiesServiceURL, queryString)
	var existingEntities []ZmonEntity

	p := napping.Params{"query": string(queryString)}.AsUrlValues()
	_, err = s.Get(existingEntitiesURL, &p, &existingEntities, nil)
	maybeAbort(err, "unable to get existing entries from ZMON")

	// delete all the existing entities
	for _, existingEntity := range existingEntities {
		deleteURL := fmt.Sprintf("%s/?id=%s", zmonEntitiesServiceURL, existingEntity.Id)
		log.Debug("about to delete zmonEntity entity with ID '%s' via calling '%s'", existingEntity.Id, deleteURL)

		p = napping.Params{"id": existingEntity.Id}.AsUrlValues()
		response, err = s.Delete(deleteURL, &p, nil, nil)
		maybeAbort(err, fmt.Sprintf("unable to delete zmonEntity with ID '%s'", existingEntity.Id))

		log.Debug("DELETE response (%d): %s", response.Status(), response.RawText())
	}

	if arguments["--onlydelete"].(bool) {
		log.Info("Option '--onlydelete' is set, exiting here.")
		os.Exit(0)
	}

	for _, datacenter := range datacenters {

		servicesURL := fmt.Sprintf("%s/services?dc=%s", consulBaseURL, datacenter)
		var services map[string][]string

		_, err := s.Get(servicesURL, nil, &services, nil)
		maybeAbort(err, fmt.Sprintf("unable to get services from Consul for DC '%s'", datacenter))

		for name, tags := range services {
			log.Info("service name: %s, service tags: %s\n", name, tags)

			nodesURL := fmt.Sprintf("%s/service/%s?dc=%s", consulBaseURL, name, datacenter)
			var nodes []Node

			_, err = s.Get(nodesURL, nil, &nodes, nil)
			maybeAbort(err, fmt.Sprintf("unable to get nodes for service %s from Consul", name))

			for _, node := range nodes {
				entity := &ZmonEntity{Type: "service"}
				entity.Id = node.ServiceID
				entity.ApplicationID = strings.Replace(node.ServiceName, ":", "-", -1) + "[techmonkeys]"
				entity.DataCenterCode = strings.ToUpper(datacenter)
				entity.Host = node.ServiceAddress
				servicePortString := strconv.Itoa(node.ServicePort)
				entity.Ports = map[string]int{
					servicePortString: node.ServicePort,
				}

				log.Debug("about to insert zmonEntity entity via calling '%s'", zmonEntitiesServiceURL)

				response, err = s.Put(zmonEntitiesServiceURL, entity, nil, nil)
				maybeAbort(err, fmt.Sprintf("unable to insert zmonEntity with ID '%s'", entity.Id))

				log.Debug("PUT response (%d): %s", response.Status(), response.RawText())
			}
		}
	}
}
