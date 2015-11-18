package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/golang/glog"
	"github.com/kr/pretty"
	"stash.zalando.net/scm/system/pmi-monitoring-connector.git/conf"
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
	var event Event
	ginCtx.BindWith(&event, binding.JSON)
	glog.Infof("marathon event: %# v", pretty.Formatter(event))
	glog.Infof("received marathon '%s' event", event.Eventtype)

	/* backends := registerBackends()
	   for _, backend := backends {
	   	// @TODO: dispatch different event types here, perhaps there is a more elegant solution...
			switch event.Eventtype {
				"api_post_event":
					ginCtx.Bindwith(&marathonEvent ApiRequest, binding.JSON)
					backend.handleEvent(marathonEvent)
				"status_update_event":
					ginCtx.Bindwith(&marathonEvent StatusUpdate, binding.JSON)
					backend.handleEvent(marathonEvent)
			}
	   }
	*/

}

func registerBackends() (error, []backend.Backend) {
	/* @TODO load available backends from /backend into an array, then iterate over it:
	for _, backend := range(backends) {
		backend.register()
	}
	return backends
	*/
}
