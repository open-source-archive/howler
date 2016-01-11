package backend

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/zalando-techmonkeys/howler/conf"
	"gopkg.in/jmcvetta/napping.v3"
)

type Zmon struct {
	name              string
	session           napping.Session
	zmonEntityService string
}

// ZmonEntity represents an entity in ZMON
// @TODO: entity.ApplicationID should be postfixed with team name which owns the service, for now it's just "[techmonkeys]"
type ZmonEntity struct {
	Type           string         `json:"type"`
	ID             string         `json:"id"`
	ApplicationID  string         `json:"application_id"`
	Host           string         `json:"host"`
	Ports          map[string]int `json:"ports"`
	DataCenterCode string         `json:"data_center_code"`
}

func (be Zmon) Name() string {
	return be.name
}

func (be Zmon) Register() (error, Backend) {

	backendConfig := conf.New().Backends["zmon"]

	s := napping.Session{}
	s.Userinfo = url.UserPassword(backendConfig["user"], backendConfig["password"])
	s.Header = &http.Header{"Content-Type": []string{"application/json"}}

	zmonEntityService := backendConfig["entityService"]

	return nil, Zmon{name: "Zmon", session: s, zmonEntityService: zmonEntityService}
}

func (be Zmon) HandleCreate(e ApiRequestEvent) {
	//TODO write implementation
}

func (be Zmon) HandleDestroy(e AppTerminatedEvent) {
	//TODO write implementation
}

func (be Zmon) HandleUpdate(e StatusUpdateEvent) {
	if e.Taskstatus == "TASK_RUNNING" {
		be.insertEntity(e)
	} else if e.Taskstatus == "TASK_KILLED" || e.Taskstatus == "TASK_LOST" { //TODO should we add more Taskstatus for when a task is killed?
		be.deleteEntity(e)
	}
	return
}

func (be Zmon) deleteEntity(e StatusUpdateEvent) error {
	var err error
	var response *napping.Response

	deleteURL := fmt.Sprintf("%s/?id=%s", be.zmonEntityService, e.Taskid)
	glog.Infof("about to delete zmonEntity entity with ID '%s' via calling '%s'", e.Taskid, deleteURL)

	p := napping.Params{"id": e.Taskid}.AsUrlValues()
	response, err = be.session.Delete(deleteURL, &p, nil, nil)
	if err != nil {
		glog.Errorf(fmt.Sprintf("unable to delete zmonEntity with ID '%s'", e.Taskid))
		return err
	}
	glog.Infof("DELETE response (%d): %s", response.Status(), response.RawText())
	return nil
}

func (be Zmon) insertEntity(e StatusUpdateEvent) error {
	var err error
	var response *napping.Response

	entity := &ZmonEntity{Type: "service"}
	entity.ID = e.Taskid
	entity.ApplicationID = e.Appid + "[techmonkeys]"
	datacenter := strings.Split(e.Host, "-")[0]
	entity.DataCenterCode = strings.ToUpper(datacenter)
	entity.Host = e.Host
	entity.Ports = make(map[string]int)
	for _, port := range e.Ports {
		entity.Ports[strconv.Itoa(port)] = port
	}

	glog.Infof("about to insert zmonEntity entity with ID '%s' via calling '%s'", e.Taskid, be.zmonEntityService)

	response, err = be.session.Put(be.zmonEntityService, entity, nil, nil)
	if err != nil {
		glog.Errorf("unable to insert zmonEntity with ID '%s'", entity.ID)
		return err
	}
	glog.Infof("PUT response (%d): %s", response.Status(), response.RawText())
	return nil
}
