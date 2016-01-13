package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/zalando-techmonkeys/howler/conf"
	"gopkg.in/jmcvetta/napping.v3"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Baboon struct {
	name          string
	session       *napping.Session
	f5PoolService string
	tokenFile     string
	domain        string
	datacenter    string
}

type BaboonService struct {
	Type                  string
	Pool                  string
	Loadbalancer          string
	PoolMember            string
	PoolMemberDescription string
	Ports                 map[int]int
}

type BaboonToken struct {
	Token string `json:"token"`
}

type addPoolMember struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type deletePoolMember struct {
	Name string `json:"name"`
}

type addPool struct {
	Name      string          `json:"name"`
	Partition string          `json:"partition"`
	Members   []addPoolMember `json:"members"`
	Monitor   string          `json:"monitor"`
}

func (be Baboon) Name() string {
	return be.name
}

func (be Baboon) Register() (error, Backend) {

	backendConfig := conf.New().Backends["baboon"]
	s := napping.Session{}
	s.Header = &http.Header{"Content-Type": []string{"application/json"}}

	f5PoolService := backendConfig["entityService"]

	return nil, Baboon{name: "baboon", session: &s, f5PoolService: f5PoolService,
		tokenFile: backendConfig["tokenFile"], domain: backendConfig["domain"],
		datacenter: backendConfig["datacenter"]}
}

func (be Baboon) getToken() string {
	var bt BaboonToken
	r, err := ioutil.ReadFile(be.tokenFile)
	if err != nil {
		glog.Errorf("can't open file, reason: %s", err)
	}
	if err := json.Unmarshal(r, &bt); err != nil {
		glog.Errorf("can't unmarshal object, reason %s", err)
	}
	return bt.Token
}

// HandleUpdate adds or removes container to loadbalancer pool
func (be Baboon) HandleUpdate(e StatusUpdateEvent) {
	be.modifyPoolMember(e)
}

// HandleCreate creates new loadbalancer pools
func (be Baboon) HandleCreate(e ApiRequestEvent) {
	be.createPool(e)
}

// HandleDestroy deletes loadbalancer pools
func (be Baboon) HandleDestroy(e AppTerminatedEvent) {
	be.destroyPool(e)
}

// destroyPoolMember calls baboon
func (be Baboon) destroyPool(e AppTerminatedEvent) {
	var (
		response *napping.Response
	)
	pool := strings.TrimLeft(e.Appid, "/")
	dcs := strings.Split(be.datacenter, ",")
	token := be.getToken()
	be.session.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	for i := range dcs {
		// needs to be more general -ltm is hardcoded which is not so great
		baboonEndpoint := fmt.Sprintf("%s%s-ltm/pools/%s", be.f5PoolService, dcs[i], pool)
		fmt.Println(baboonEndpoint)
		u, err := url.Parse(baboonEndpoint)
		if err != nil {
			glog.Errorf("unable to parse rawurl, reason %s", err)
		}
		glog.Infof("about to remove F5 pool entity with AppID '%s' via calling '%s'", e.Appid, baboonEndpoint)

		response, err = be.session.Delete(u.String(), nil, nil, nil)
		if err != nil {
			glog.Errorf("unable to remove pool '%s'", pool)
		}
		glog.Infof("DELETE response (%d): %s", response.Status(), response.RawText())
	}
}

// createPoolMember calls baboon
func (be Baboon) createPool(e ApiRequestEvent) {
	var (
		response *napping.Response
	)

	pool := strings.TrimLeft(e.Appdefinition.ID, "/")
	dcs := strings.Split(be.datacenter, ",")
	token := be.getToken()
	be.session.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	payload := addPool{Name: pool, Monitor: "tcp"}
	for i := range dcs {
		// needs to be more general -ltm is hardcoded which is not so great
		baboonEndpoint := fmt.Sprintf("%s%s-ltm/pools", be.f5PoolService, dcs[i])
		fmt.Println(baboonEndpoint)
		u, err := url.Parse(baboonEndpoint)
		if err != nil {
			glog.Errorf("unable to parse rawurl, reason %s", err)
		}
		glog.Infof("about to add F5 pool entity with AppID '%s' via calling '%s'", e.Appdefinition.ID, baboonEndpoint)

		response, err = be.session.Post(u.String(), payload, nil, nil)
		if err != nil {
			glog.Errorf("unable to add pool '%s'", pool)
		}
		glog.Infof("POST response (%d): %s", response.Status(), response.RawText())
	}
}

// modifyPoolMember calls baboon
func (be Baboon) modifyPoolMember(e StatusUpdateEvent) {
	var (
		response *napping.Response
		entity   BaboonService
	)
	entity.Ports = make(map[int]int)
	for i, port := range e.Ports {
		entity.Ports[i] = port
	}

	entity.Pool = strings.TrimLeft(e.Appid, "/")
	dc := strings.Split(e.Host, "-")[0]
	entity.Loadbalancer = fmt.Sprintf("%s-ltm", dc)

	host := strings.Split(e.Host, ".")[0]
	host = fmt.Sprintf("%s.%s", host, be.domain)
	ip, err := net.LookupHost(host)
	if err != nil {
		glog.Errorf("unable to lookup host %s", host)
	}
	entity.PoolMember = fmt.Sprintf("%s:%s", ip[0], strconv.Itoa(entity.Ports[0]))

	token := be.getToken()
	be.f5PoolService = fmt.Sprintf("%s%s/pools/%s/members", be.f5PoolService, entity.Loadbalancer, entity.Pool)
	u, err := url.Parse(be.f5PoolService)
	if err != nil {
		glog.Errorf("unable to parse rawurl, reason %s", err)
	}
	glog.Infof("about to modify F5 pool member entity with TaskID '%s' via calling '%s'", e.Taskid, be.f5PoolService)

	switch {
	case e.Taskstatus == "TASK_RUNNING":
		entity.Type = "Add pool member"
		be.session.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		response, err = be.session.Post(u.String(), addPoolMember{Name: entity.PoolMember,
			Description: entity.PoolMemberDescription}, nil, nil)
		if err != nil {
			glog.Errorf("unable to add pool member '%s', reason: %s", entity.PoolMember, err)
		}
		glog.Infof("POST response (%d): %s", response.Status(), response.RawText())
	case e.Taskstatus == "TASK_KILLED":
		// napping doesn't support payload for DELETE methods
		// using plain http client to delete pool member
		entity.Type = "Delete pool member"
		payload := deletePoolMember{Name: entity.PoolMember}
		buf, err := json.Marshal(payload)
		if err != nil {
			glog.Errorf("can not marshal entity, reason %s", err)
		}

		req, err := http.NewRequest("DELETE", be.f5PoolService, bytes.NewBuffer(buf)) // <-- URL-encoded payload
		if err != nil {
			glog.Errorf("unable make a new request, reason: %s", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Add("Content-Type", "application/json")
		c := &http.Client{}
		rsp, err := c.Do(req)
		defer rsp.Body.Close()
		if err != nil {
			glog.Errorf("unable to remove pool member '%s'", entity.PoolMember)
		}
		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			glog.Errorf("unable to read response body, reason %s", err)
		}
		glog.Infof("DELETE response (%s): %s", rsp.Status, string(body))
	default:
		entity.Type = "Unknown type"
		glog.Errorf(entity.Type)
	}
}
