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

	EventType := determineEventType(ginCtx.Request)

	for _, backendImplementation := range registeredBackends {
		// dispatching event types here, @TODO: perhaps there is a more elegant solution...

		switch EventType {
		case "api_post_event":
			var marathonEvent backend.ApiRequestEvent
			ginCtx.Bind(&marathonEvent)
			backendImplementation.HandleEvent(marathonEvent)
		case "status_update_event":
			var marathonEvent backend.StatusUpdateEvent
			ginCtx.Bind(&marathonEvent)
			backendImplementation.HandleEvent(marathonEvent)
		default:
			glog.Errorf("event type '%s' is not dispatched to any backend", EventType)
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
