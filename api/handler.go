package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/golang/glog"
	"github.com/kr/pretty"
	"stash.zalando.net/scm/system/pmi-monitoring-connector.git/backend"
	"stash.zalando.net/scm/system/pmi-monitoring-connector.git/conf"
)

var (
	enabledBackends = []backend.Backend {backend.Zmon2{}}
)

func rootHandler(ginCtx *gin.Context) {
	config := conf.New()
	ginCtx.JSON(http.StatusOK, gin.H{"pmi-monitoring-connector": fmt.Sprintf("Build Time: %s - Git Commit Hash: %s", config.VersionBuildStamp, config.VersionGitHash)})
}

func getStatus(ginCtx *gin.Context) {
	glog.Info("basic health check, will always return 'OK' for now")
	ginCtx.String(http.StatusOK, "OK")
}

func dispatchEventType(ginCtx *gin.Context) error

// endpoint for receiving marathon event bus messages
func createEvent(ginCtx *gin.Context) {

	ginCtx.Request.ParseForm()

	var event backend.Event
	ginCtx.BindWith(&event, binding.JSON)
	glog.Infof("marathon event: %# v", pretty.Formatter(event))
	glog.Infof("received marathon '%s' event", event.Eventtype)

	err, backends := registerBackends()
	for _, backendImplementation := range backends {
		// dispatching event types here, @TODO: perhaps there is a more elegant solution...
		switch event.Eventtype {
		case "api_post_event":
			var marathonEvent backend.ApiRequest
			ginCtx.BindWith(marathonEvent, binding.JSON)
			backendImplementation.HandleEvent(marathonEvent)
		case "status_update_event":
			var marathonEvent backend.StatusUpdate
			ginCtx.BindWith(marathonEvent, binding.JSON)
			backendImplementation.HandleEvent(marathonEvent)
		default:
			glog.Errorf("event type '%s' is not dispatched to any backend", event.Eventtype)
		}
	}

}

func registerBackends() (error, []backend.Backend) {

	var backends []backend.Backend
	for _, backendImplementation := range enabledBackends {
		err, backendInstance := backendImplementation.Register()
		if err != nil {
			return err, nil
		}
		backends = append(backends, backendInstance)
	}
	return nil, backends
}
