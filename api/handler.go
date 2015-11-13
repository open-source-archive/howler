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

// endpoint for receiving marathon event bus messages
func createEvent(ginCtx *gin.Context) {
	// @TODO: dispatch different backends here...
    ginCtx.Request.ParseForm()
    var event StatusUpdate
    ginCtx.BindWith(&event, binding.JSON)
    glog.Infof("marathon event: %# v", pretty.Formatter(event))
}
