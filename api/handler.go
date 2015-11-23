package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/kr/pretty"
	"stash.zalando.net/scm/system/pmi-monitoring-connector.git/backend"
	"stash.zalando.net/scm/system/pmi-monitoring-connector.git/conf"
)

// @TODO make enabledBackends configurable
var (
	enabledBackends    = []backend.Backend{backend.Zmon{}, backend.DummyBackend{}}
	registeredBackends = registerBackends()
)

// rootHandler serving "/" which returns build information
func rootHandler(ginCtx *gin.Context) {
	config := conf.New()
	ginCtx.JSON(http.StatusOK, gin.H{"pmi-monitoring-connector": fmt.Sprintf("Build Time: %s - Git Commit Hash: %s", config.VersionBuildStamp, config.VersionGitHash)})
}

// basic health check, will always return 'OK' for now
func getStatus(ginCtx *gin.Context) {
	ginCtx.String(http.StatusOK, "OK")
}

// endpoint for receiving marathon event bus messages
func createEvent(ginCtx *gin.Context) {

	eventType := determineEventType(ginCtx.Request)

	// dispatching event types here

	switch eventType {
	case "api_post_event":
		var marathonEvent backend.ApiRequestEvent
		ginCtx.Bind(&marathonEvent)

		glog.Infof("dispatching to backends: %# v", pretty.Formatter(marathonEvent))
		for _, backendImplementation := range registeredBackends {
			glog.Infof("dispatching event to backend '%s'", backendImplementation.Name())
			backendImplementation.HandleEvent(marathonEvent)
		}
	case "status_update_event":
		var marathonEvent backend.StatusUpdateEvent
		ginCtx.Bind(&marathonEvent)

		glog.Infof("dispatching to backends: %# v", pretty.Formatter(marathonEvent))
		for _, backendImplementation := range registeredBackends {
			glog.Infof("dispatching event to backend '%s'", backendImplementation.Name())
			backendImplementation.HandleEvent(marathonEvent)
		}
	default:
		msg := fmt.Sprintf("event type '%s' is not dispatched to any backend", eventType)
		glog.Errorf(msg)
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	ginCtx.JSON(http.StatusOK, gin.H{"result": "Success"})
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

func determineEventType(r *http.Request) string {

	// temporary buffer
	b := bytes.NewBuffer(make([]byte, 0))

	// TeeReader returns a Reader that writes to b what it reads from r.Body.
	reader := io.TeeReader(r.Body, b)

	var event backend.Event
	if err := json.NewDecoder(reader).Decode(&event); err != nil {
		glog.Fatal(err)
	}

	// we are done with body
	defer r.Body.Close()

	// NopCloser returns a ReadCloser with a no-op Close method wrapping the provided Reader r.
	r.Body = ioutil.NopCloser(b)

	return event.Eventtype
}
