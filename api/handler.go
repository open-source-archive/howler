package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/kr/pretty"
	"stash.zalando.net/scm/system/pmi-monitoring-connector.git/backend"
	"stash.zalando.net/scm/system/pmi-monitoring-connector.git/conf"
)

var (
	enabledBackends    = []backend.Backend{backend.Zmon2{}}
	registeredBackends = registerBackends()
)

func rootHandler(ginCtx *gin.Context) {
	config := conf.New()
	ginCtx.JSON(http.StatusOK, gin.H{"pmi-monitoring-connector": fmt.Sprintf("Build Time: %s - Git Commit Hash: %s", config.VersionBuildStamp, config.VersionGitHash)})
}

func getStatus(ginCtx *gin.Context) {
	glog.Info("basic health check, will always return 'OK' for now")
	ginCtx.String(http.StatusOK, "OK")
}

// endpoint for receiving marathon event bus messages
func createEvent(ginCtx *gin.Context) {

	body, _ := ioutil.ReadAll(ginCtx.Request.Body)
	var event backend.Event
	err := json.Unmarshal(body, &event)
	if err != nil {
		// @TODO: better error handling
		glog.Errorf("Unable to decode event body: %s\n", err.Error())
		return
	}
	glog.Infof("received marathon '%s' event: %# v", event.Eventtype, pretty.Formatter(event))

	for _, backendImplementation := range registeredBackends {
		// dispatching event types here, @TODO: perhaps there is a more elegant solution...
		switch event.Eventtype {
		case "api_post_event":
			var marathonEvent backend.ApiRequestEvent
			json.Unmarshal(body, &marathonEvent)
			backendImplementation.HandleEvent(marathonEvent)
		case "status_update_event":
			var marathonEvent backend.StatusUpdateEvent
			json.Unmarshal(body, &marathonEvent)
			backendImplementation.HandleEvent(marathonEvent)
		default:
			glog.Errorf("event type '%s' is not dispatched to any backend", event.Eventtype)
		}
	}

	content := gin.H{"result": "Success"}
	ginCtx.JSON(200, content)
}

func registerBackends() []backend.Backend {

	var backends []backend.Backend
	for _, backendImplementation := range enabledBackends {
		err, backendInstance := backendImplementation.Register()
		if err != nil {
			glog.Fatalf("unable to register backend %s", backendImplementation)
		}
		backends = append(backends, backendInstance)
	}
	return backends
}
