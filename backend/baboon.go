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
	"sync"
)

// Baboon is the basic type
type Baboon struct {
	config  map[string]string
	session *napping.Session
}

// LTMPoolService is the basic pool type
// to create/modify/delete pools and members
type LTMPoolService struct {
	Type                  string
	Pool                  string
	Loadbalancer          string
	PoolMember            string
	PoolMemberDescription string
	Ports                 map[int]int
}

// BaboonToken inherits the token
// to call baboon-proxy
type BaboonToken struct {
	Token string `json:"token"`
}

// addLTMPoolMember inherits fields
// to add a member in a pool LTM
type addLTMPoolMember struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// addLTMPoolMember inherits fields
// to add a member in a pool LTM
type deleteLTMPoolMember struct {
	Name string `json:"name"`
}

// addLTMPool inherits fields
// to add a pool LTM
type addLTMPool struct {
	Name      string             `json:"name"`
	Partition string             `json:"partition"`
	Members   []addLTMPoolMember `json:"members"`
	Monitor   string             `json:"monitor"`
}

// addGTMPool inherits fields
// to add a pool GTM
type addGTMPool struct {
	Name    string             `json:"name"`
	Members []addGTMPoolMember `json:"members"`
	Monitor string             `json:"monitor"`
}

// addGTMWideip inherits fields
// to add a wideip GTM
type addGTMWideip struct {
	Name       string             `json:"name"`
	Pools      []addGTMWideIPPool `json:"pools"`
	PoolLBMode string             `json:"poolLBMode"`
}

// addGTMWideIPPool inherits fields
// to add a pool in a wideip GTM
type addGTMWideIPPool struct {
	Name string `json:"name"`
}

// addGTMPoolMember inherits fields
// to add a member in a pool GTM
type addGTMPoolMember struct {
	Name         string `json:"name"`
	Loadbalancer string `json:"loadbalancer"`
}

//Name returns the backend service name
func (be Baboon) Name() string {
	return be.config["name"]
}

// Register reads backend config for baboon
func (be Baboon) Register() (error, Backend) {
	config := conf.New().Backends["baboon"]
	glog.Infof("%+v", config)
	glog.Infof("%s", config["tokenFile"])
	s := napping.Session{}
	s.Header = &http.Header{"Content-Type": []string{"application/json"}}
	backendConfig := &Baboon{config: config, session: &s}
	return nil, backendConfig
}

// getToken reads from tokenFile
func (be Baboon) getToken() string {
	var bt BaboonToken
	r, err := ioutil.ReadFile(be.config["tokenFile"])
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
	be.modify(e)
}

// HandleCreate creates new LTM pools, GTM pools and GTM wideip
func (be Baboon) HandleCreate(e ApiRequestEvent) {
	be.create(e)
}

// HandleDestroy deletes LTM pools, GTM pools and GTM wideip
func (be Baboon) HandleDestroy(e AppTerminatedEvent) {
	be.destroy(e)
}

// destroy calls baboon-proxy to destroy LTM pools, GTM pool and GTM wideip
func (be Baboon) destroy(e AppTerminatedEvent) {
	var (
		response *napping.Response
		wait     sync.WaitGroup
	)
	appName := strings.TrimLeft(e.Appid, "/")
	poolName := fmt.Sprintf("%s%s", be.config["ltmPoolPrefix"], appName)
	loadbalancerSlice := strings.Split(be.config["loadbalancer"], ",")
	token := be.getToken()
	be.session.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	wait.Add(len(loadbalancerSlice))
	for i := range loadbalancerSlice {
		// running multiple go routines to delete LTM pools concurrently
		// otherwise it's to slow waiting for each LTM
		go be.destroyLTMPool(loadbalancerSlice[i], e, poolName, &wait)
	}
	// wait for destroying all LTM pools
	wait.Wait()
	// calls baboon-proxy to delete GTM wideip and pool
	baboonGTMEndpoint := fmt.Sprintf("%s%s/wideips/%s.%s",
		be.config["entityGTMService"], be.config["trafficManager"], appName, be.config["gtmDomain"])
	u, err := url.Parse(baboonGTMEndpoint)
	if err != nil {
		glog.Errorf("unable to parse rawurl, reason %s", err)
		return
	}
	glog.Infof("about to delete F5 GTM wideip entity with AppID '%s' via calling '%s'", e.Appid, baboonGTMEndpoint)

	response, err = be.session.Delete(u.String(), nil, nil, nil)
	if err != nil {
		glog.Errorf("unable to delete GTM wideip '%s.%s'", appName, be.config["gtmDomain"])
		return
	}
	glog.Infof("DELETE response (%d): %s", response.Status(), response.RawText())
	baboonGTMEndpoint = fmt.Sprintf("%s%s/pools/%s",
		be.config["entityGTMService"], be.config["trafficManager"], poolName)
	u, err = url.Parse(baboonGTMEndpoint)
	if err != nil {
		glog.Errorf("unable to parse rawurl, reason %s", err)
		return
	}
	glog.Infof("about to delete F5 GTM pool entity with AppID '%s' via calling '%s'", e.Appid, baboonGTMEndpoint)

	response, err = be.session.Delete(u.String(), nil, nil, nil)
	if err != nil {
		glog.Errorf("unable to add GTM pool '%s'", poolName)
		return
	}
	glog.Infof("DELETE response (%d): %s", response.Status(), response.RawText())
}

// destroyLTMPool calls baboon-proxy destroying all pools in all DCs concurrently
func (be Baboon) destroyLTMPool(loadbalancer string, e AppTerminatedEvent, poolName string, wait *sync.WaitGroup) {
	defer wait.Done()
	baboonEndpoint := fmt.Sprintf("%s%s/pools/%s",
		be.config["entityLTMService"], loadbalancer, poolName)
	u, err := url.Parse(baboonEndpoint)
	if err != nil {
		glog.Errorf("unable to parse rawurl, reason %s", err)
		return
	}
	glog.Infof("about to remove F5 pool entity with AppID '%s' via calling '%s'", e.Appid, baboonEndpoint)

	response, err := be.session.Delete(u.String(), nil, nil, nil)
	if err != nil {
		glog.Errorf("unable to remove pool '%s'", poolName)
		return
	}
	glog.Infof("DELETE response (%d): %s", response.Status(), response.RawText())
	return
}

// create calls baboon-proxy to create LTM pools, GTM pool and GTM wideip
func (be Baboon) create(e ApiRequestEvent) {
	var (
		response *napping.Response
		wait     sync.WaitGroup
	)

	appName := strings.TrimLeft(e.Appdefinition.ID, "/")
	poolName := fmt.Sprintf("%s%s", be.config["ltmPoolPrefix"], appName)
	loadbalancerSlice := strings.Split(be.config["loadbalancer"], ",")
	virtualServerSlice := strings.Split(be.config["virtualServer"], ",")
	fmt.Println(loadbalancerSlice)
	token := be.getToken()
	be.session.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	payloadLTM := addLTMPool{Name: poolName, Monitor: be.config["ltmPoolMonitor"]}
	payloadGTMPool := addGTMPool{}
	payloadGTMPool.Name = poolName
	payloadGTMPool.Monitor = be.config["gtmPoolMonitor"]
	for i := range loadbalancerSlice {
		payloadGTMPool.Members = append(payloadGTMPool.Members, addGTMPoolMember{
			Name: virtualServerSlice[i], Loadbalancer: loadbalancerSlice[i],
		})
	}
	payloadGTMWideip := addGTMWideip{}
	payloadGTMWideip.Name = fmt.Sprintf("%s.%s", appName, be.config["gtmDomain"])
	payloadGTMWideip.Pools = append(payloadGTMWideip.Pools, addGTMWideIPPool{Name: poolName})
	payloadGTMWideip.PoolLBMode = be.config["gtmWideipMonitor"]

	wait.Add(len(loadbalancerSlice))
	for i := range loadbalancerSlice {
		// running multiple go routines to create LTM pools concurrently
		// otherwise it's to slow for incoming status_update_events
		// LTM pool members can only be modified if the LTM pool already exists
		go be.createLTMPool(loadbalancerSlice[i], e, poolName, payloadLTM, &wait)
	}
	// wait for creating all LTM pools
	wait.Wait()
	// calls baboon-proxy to create GTM pool and wideip
	baboonGTMEndpoint := fmt.Sprintf("%s%s/pools", be.config["entityGTMService"],
		be.config["trafficManager"])
	u, err := url.Parse(baboonGTMEndpoint)
	if err != nil {
		glog.Errorf("unable to parse rawurl, reason %s", err)
		return
	}
	glog.Infof("about to add F5 GTM pool entity with AppID '%s' via calling '%s'", e.Appdefinition.ID, baboonGTMEndpoint)

	response, err = be.session.Post(u.String(), payloadGTMPool, nil, nil)
	if err != nil {
		glog.Errorf("unable to add GTM pool '%s'", poolName)
		return
	}
	glog.Infof("POST response (%d): %s", response.Status(), response.RawText())
	baboonGTMEndpoint = fmt.Sprintf("%s%s/wideips", be.config["entityGTMService"],
		be.config["trafficManager"])
	u, err = url.Parse(baboonGTMEndpoint)
	if err != nil {
		glog.Errorf("unable to parse rawurl, reason %s", err)
		return
	}
	glog.Infof("about to add F5 GTM wideip entity with AppID '%s' via calling '%s'", e.Appdefinition.ID, baboonGTMEndpoint)

	response, err = be.session.Post(u.String(), payloadGTMWideip, nil, nil)
	if err != nil {
		glog.Errorf("unable to create GTM wideip '%s'")
		return
	}
	glog.Infof("POST response (%d): %s", response.Status(), response.RawText())
}

// createLTMPool calls baboon-proxy creating all pools in all DCs concurrently
func (be Baboon) createLTMPool(loadbalancer string, e ApiRequestEvent, poolName string, payloadLTM addLTMPool, wait *sync.WaitGroup) {
	defer wait.Done()
	baboonLTMEndpoint := fmt.Sprintf("%s%s/pools", be.config["entityLTMService"], loadbalancer)
	u, err := url.Parse(baboonLTMEndpoint)
	if err != nil {
		glog.Errorf("unable to parse rawurl, reason %s", err)
		return
	}
	glog.Infof("about to add F5 LTM pool entity with AppID '%s' via calling '%s'", e.Appdefinition.ID, baboonLTMEndpoint)

	response, err := be.session.Post(u.String(), payloadLTM, nil, nil)
	if err != nil {
		glog.Errorf("unable to add LTM pool '%s'", poolName)
		return
	}
	glog.Infof("POST response (%d): %s", response.Status(), response.RawText())
	return
}

// modify calls baboon-proxy to add or delete members in LTM pools
func (be Baboon) modify(e StatusUpdateEvent) {
	var (
		response *napping.Response
		entity   LTMPoolService
	)
	entity.Ports = make(map[int]int)
	for i, port := range e.Ports {
		entity.Ports[i] = port
	}

	entity.Pool = fmt.Sprintf("%s%s",
		be.config["ltmPoolPrefix"], strings.TrimLeft(e.Appid, "/"))
	dc := strings.Split(e.Host, "-")[0]
	entity.Loadbalancer = fmt.Sprintf("%s-ltm", dc)

	host := strings.Split(e.Host, ".")[0]
	host = fmt.Sprintf("%s.%s", host, be.config["domain"])
	ip, err := net.LookupHost(host)
	if err != nil {
		glog.Errorf("unable to lookup host %s", host)
		return
	}
	entity.PoolMember = fmt.Sprintf("%s:%s", ip[0], strconv.Itoa(entity.Ports[0]))

	token := be.getToken()
	urlLTMMembers := fmt.Sprintf("%s%s/pools/%s/members", be.config["entityLTMService"],
		entity.Loadbalancer, entity.Pool)
	u, err := url.Parse(urlLTMMembers)
	if err != nil {
		glog.Errorf("unable to parse rawurl, reason %s", err)
		return
	}
	glog.Infof("about to modify F5 pool member entity with TaskID '%s' via calling '%s'",
		e.Taskid, u.String())

	switch {
	case e.Taskstatus == "TASK_RUNNING":
		entity.Type = "Add pool member"
		be.session.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		response, err = be.session.Post(u.String(), addLTMPoolMember{Name: entity.PoolMember,
			Description: entity.PoolMemberDescription}, nil, nil)
		if err != nil {
			glog.Errorf("unable to add pool member '%s', reason: %s", entity.PoolMember, err)
			return
		}
		glog.Infof("POST response (%d): %s", response.Status(), response.RawText())
	case e.Taskstatus == "TASK_KILLED":
		// napping doesn't support payload for DELETE methods
		// using plain http client to delete pool member
		entity.Type = "Delete pool member"
		payload := deleteLTMPoolMember{Name: entity.PoolMember}
		buf, err := json.Marshal(payload)
		if err != nil {
			glog.Errorf("can not marshal entity, reason %s", err)
			return
		}

		req, err := http.NewRequest("DELETE", u.String(),
			bytes.NewBuffer(buf)) // <-- URL-encoded payload
		if err != nil {
			glog.Errorf("unable make a new request, reason: %s", err)
			return
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")
		c := &http.Client{}
		rsp, err := c.Do(req)
		if rsp != nil {
			defer rsp.Body.Close()
		}
		if err != nil {
			glog.Errorf("unable to remove pool member '%s'", entity.PoolMember)
			return
		}
		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			glog.Errorf("unable to read response body, reason %s", err)
			return
		}
		glog.Infof("DELETE response (%s): %s", rsp.Status, string(body))
	default:
		entity.Type = "Unknown type"
		glog.Errorf(entity.Type)
	}
}
