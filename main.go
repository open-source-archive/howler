package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	//    "github.com/vrischmann/envconfig"
)

// @TODO: entity.ApplicationID should be postfixed with team name which owns the service, for now it's just "[techmonkeys]"

/*
examples for entities:
id=consul, application_id=consul[techmonkeys], host=gth-consul02, data_center_code=GTH

id=gth-ltm01[Platform/System] host=10.228.116.140 team=Platform/System type=loadbalancer data_center_code=GTH
gth-gtm01[Platform/System] gtm_listeners data_center_code=GTH host=62.138.84.246 team=Platform/System
gth-gtm02[Platform/System] gtm_listeners data_center_code=GTH host=62.138.84.254 team=Platform/System
gth-itm01[Platform/System] gtm_listeners data_center_code=GTH host=10.64.112.8 team=Platform/System
gth-itm02[Platform/System] gtm_listeners data_center_code=GTH host=10.64.112.9 team=Platform/System
*/

var usage = fmt.Sprintf(`
Usage:
    %s [options]

Options:
    --custom    this is my custom flag

Common Options:
  -h, --help            show this help message and exit
  -v, --verbose         Increase verbosity level (show debug messages)
  -q, --quiet           Decrease verbosity level
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
	Type           string            `json:"type"`
	Id             string            `json:"id"`
	ApplicationID  string            `json:"application_id"`
	Host           string            `json:"host"`
	Ports          map[string]string `json:"ports"`
	DataCenterCode string            `json:"data_center_code"`
}

func httpGet(url string) ([]byte, error) {

	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(viper.GetString("user"), viper.GetString("password"))

	body, err := makeRequest(req, url)
	if err != nil {
		log.Warning("failed to make GET request to %s", url)
	}
	return body, err
}

func httpPut(url string, data []byte) ([]byte, error) {

	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(viper.GetString("user"), viper.GetString("password"))

	body, err := makeRequest(req, url)
	if err != nil {
		log.Warning("failed to make PUT request to %s", url)
	}
	return body, err
}

func httpDelete(url string) ([]byte, error) {

	req, _ := http.NewRequest("DELETE", url, nil)
	req.SetBasicAuth(viper.GetString("user"), viper.GetString("password"))

	body, err := makeRequest(req, url)
	if err != nil {
		log.Warning("failed to make PUT request to %s", url)
	}
	return body, err
}

func makeRequest(req *http.Request, url string) ([]byte, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	return body, err
}

func maybeAbort(err error, msg string) {
	if err != nil {
		log.Fatalf("ERROR: %s %+v", msg, err)
	}
}

func readConfig(cf string) {

	viper.SetConfigType("yaml")
	viper.SetConfigName(cf)
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath(fmt.Sprintf("%s/.config/", os.ExpandEnv("$HOME")))

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("unable to read config file: %s", err)
	}
}

func notImplemented(option string) {
	fmt.Printf("Option %s is not implemented yet.\n", option)
}

func main() {

	arguments, err := docopt.Parse(usage, nil, true, fmt.Sprintf("%s 0.1-dev", os.Args[0]), false)
	if err != nil {
		panic("Could not parse CLI")
	}

	log.Debug("log: %+v\n", arguments)

	if arguments["--config"] != nil {
		notImplemented("--config")
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

	zmonEntitiesServiceURL := ZMON_HOST + ZMON_URL
	consulBaseURL := fmt.Sprintf("https://%s:8500/v1/catalog", CONSUL_MASTER)
	datacenters := [...]string{"gth", "itr"}

	readConfig("zmon-connector")

	// get all existing entities from ZMON
	query := map[string]string{"type": "service"}
	queryString, _ := json.Marshal(query)

	existingEntitiesURL := fmt.Sprintf("%s/?query=%s", zmonEntitiesServiceURL, queryString)
	var existingEntities []ZmonEntity

	response, err := httpGet(existingEntitiesURL)
	maybeAbort(err, "unable to get existing entries from ZMON")

	err = json.Unmarshal(response, &existingEntities)
	maybeAbort(err, "failed to unmarshal data from "+existingEntitiesURL+" to struct:")

	// delete all the existing entities
	for _, existingEntity := range existingEntities {
		response, err := httpDelete(fmt.Sprintf("%s/?id=%s", zmonEntitiesServiceURL, existingEntity.Id))
		maybeAbort(err, fmt.Sprintf("unable to delete zmonEntity with id '%s'", existingEntity.Id))
		log.Debug("response: %+v", string(response))
	}

	for _, datacenter := range datacenters {

		servicesURL := fmt.Sprintf("%s/services?dc=%s", consulBaseURL, datacenter)
		var services map[string][]string

		response, err := httpGet(servicesURL)
		maybeAbort(err, fmt.Sprintf("unavble to get services from Consul for DC '%s'", datacenter))

		err = json.Unmarshal(response, &services)
		maybeAbort(err, "failed to unmarshal data from "+servicesURL+" to struct:")

		for name, tags := range services {
			log.Info("service name: %s, service tags: %s\n", name, tags)

			nodesURL := fmt.Sprintf("%s/service/%s?dc=%s", consulBaseURL, name, datacenter)
			var nodes []Node

			response, err := httpGet(nodesURL)
			maybeAbort(err, fmt.Sprintf("unable to get nodes for service %s from Consul", name))

			err = json.Unmarshal(response, &nodes)
			maybeAbort(err, fmt.Sprintf("failed to unmarshal data from %s to struct", nodesURL))

			for _, node := range nodes {
				entity := &ZmonEntity{Type: "service"}
				entity.Id = node.ServiceID
				entity.ApplicationID = strings.Replace(node.ServiceName, ":", "-", -1) + "[techmonkeys]"
				entity.DataCenterCode = strings.ToUpper(datacenter)
				entity.Host = node.ServiceAddress
				servicePortString := strconv.Itoa(node.ServicePort)
				entity.Ports = map[string]string{
					servicePortString: servicePortString,
				}

				jsonString, err := json.Marshal(entity)
				maybeAbort(err, "")
				log.Debug(string(jsonString))

				response, err := httpPut(zmonEntitiesServiceURL, jsonString)
				log.Debug("response: %+v", string(response))
			}
		}
	}
}
