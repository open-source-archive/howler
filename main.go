package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/zalando-techmonkeys/zalando-cli"
	"gopkg.in/jmcvetta/napping.v3"
)

// @TODO: entity.ApplicationID should be postfixed with team name which owns the service, for now it's just "[techmonkeys]"

const (
	zmonHost     = "https://zmon2.zalando.net"
	zmonURL      = "/rest/api/v1/entities/"
	consulMaster = "gth-consul01.zalando"
)

var (
	binary  = path.Base(os.Args[0])
	usage = fmt.Sprintf(`
Usage:
    %s [options]

`, binary)
)

// Node represents a node as it is known in Consul
type Node struct {
	Node           string
	Address        string
	ServiceID      string
	ServiceName    string
	ServiceTags    []string
	ServiceAddress string
	ServicePort    int
}

// ZmonEntity represents an entity in ZMON
type ZmonEntity struct {
	Type           string         `json:"type"`
	ID             string         `json:"id"`
	ApplicationID  string         `json:"application_id"`
	Host           string         `json:"host"`
	Ports          map[string]int `json:"ports"`
	DataCenterCode string         `json:"data_center_code"`
}

func maybeAbort(err error, msg string) {
	if err != nil {
		cli.Log.Fatalf("ERROR: %s %+v", msg, err)
	}
}

func notImplemented(option string) {
	fmt.Printf("Option %s is not implemented yet.\n", option)
}

func main() {
	var err error
	var response *napping.Response

	arguments := cli.Configure(usage)

	zmonEntitiesServiceURL := zmonHost + zmonURL
	consulBaseURL := fmt.Sprintf("https://%s:8500/v1/catalog", consulMaster)
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
	cli.Log.Info("deleting %d existing entities from ZMON", len(existingEntities))
	for _, existingEntity := range existingEntities {
		deleteURL := fmt.Sprintf("%s/?id=%s", zmonEntitiesServiceURL, existingEntity.ID)
		cli.Log.Debug("about to delete zmonEntity entity with ID '%s' via calling '%s'", existingEntity.ID, deleteURL)

		p = napping.Params{"id": existingEntity.ID}.AsUrlValues()
		response, err = s.Delete(deleteURL, &p, nil, nil)
		maybeAbort(err, fmt.Sprintf("unable to delete zmonEntity with ID '%s'", existingEntity.ID))

		cli.Log.Debug("DELETE response (%d): %s", response.Status(), response.RawText())
	}

	if arguments["--onlydelete"].(bool) {
		cli.Log.Info("Option '--onlydelete' is set, exiting here.")
		os.Exit(0)
	}

	for _, datacenter := range datacenters {

		servicesURL := fmt.Sprintf("%s/services?dc=%s", consulBaseURL, datacenter)
		var services map[string][]string

		_, err := s.Get(servicesURL, nil, &services, nil)
		maybeAbort(err, fmt.Sprintf("unable to get services from Consul for DC '%s'", datacenter))

		for name, tags := range services {

			nodesURL := fmt.Sprintf("%s/service/%s?dc=%s", consulBaseURL, name, datacenter)
			var nodes []Node

			_, err = s.Get(nodesURL, nil, &nodes, nil)
			maybeAbort(err, fmt.Sprintf("unable to get nodes for service %s from Consul", name))

			cli.Log.Info("syncing service '%s' (tags: %s) with %d nodes\n", name, tags, len(nodes))
			for _, node := range nodes {
				entity := &ZmonEntity{Type: "service"}
				entity.ID = node.ServiceID
				entity.ApplicationID = strings.Replace(node.ServiceName, ":", "-", -1) + "[techmonkeys]"
				entity.DataCenterCode = strings.ToUpper(datacenter)
				entity.Host = node.ServiceAddress
				servicePortString := strconv.Itoa(node.ServicePort)
				entity.Ports = map[string]int{
					servicePortString: node.ServicePort,
				}

				cli.Log.Debug("about to insert zmonEntity entity via calling '%s'", zmonEntitiesServiceURL)

				response, err = s.Put(zmonEntitiesServiceURL, entity, nil, nil)
				maybeAbort(err, fmt.Sprintf("unable to insert zmonEntity with ID '%s'", entity.ID))

				cli.Log.Debug("PUT response (%d): %s", response.Status(), response.RawText())
			}
		}
	}
}
