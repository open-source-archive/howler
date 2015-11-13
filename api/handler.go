package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
    "stash.zalando.net/scm/system/pmi-monitoring-connector.git/conf"
)

func rootHandler(ginCtx *gin.Context) {
	config := conf.New()
	ginCtx.JSON(http.StatusOK, gin.H{"pmi-monitoring-connector": fmt.Sprintf("Build Time: %s - Git Commit Hash: %s", config.VersionBuildStamp, config.VersionGitHash)})
}

func health(ginCtx *gin.Context) {
    glog.Info("basic health check, will always return 'OK' for now")
    ginCtx.JSON(http.StatusOK, gin.H{"status": "OK"})
}

