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
	"github.com/zalando-techmonkeys/howler/backend"
	"github.com/zalando-techmonkeys/howler/backendconfig"
	"github.com/zalando-techmonkeys/howler/conf"
)

// rootHandler serving "/" which returns build information
func rootHandler(ginCtx *gin.Context) {
	config := conf.New()
	ginCtx.JSON(http.StatusOK, gin.H{"howler": fmt.Sprintf("Version: %s - Build Time: %s - Git Commit Hash: %s", config.Version, config.BuildStamp, config.GitHash)})
}

// basic health check, will always return 'OK' for now
func getStatus(ginCtx *gin.Context) {
	ginCtx.String(http.StatusOK, "OK")
}

// endpoint for receiving marathon event bus messages
// Plugins will get notified in a goroutine.
func createEvent(ginCtx *gin.Context) {

	eventType := determineEventType(ginCtx.Request)

	// dispatching event types here

	switch eventType {
	case "api_post_event":
		var marathonEvent backend.ApiRequestEvent
		ginCtx.Bind(&marathonEvent)

		glog.Infof("dispatching to backends: %# v", pretty.Formatter(marathonEvent))
		for _, backendImplementation := range backendconfig.RegisteredBackends {
			glog.Infof("dispatching event to backend '%s'", backendImplementation.Name())
			go backendImplementation.HandleCreate(marathonEvent)
		}
	case "status_update_event":
		var marathonEvent backend.StatusUpdateEvent
		ginCtx.Bind(&marathonEvent)

		glog.Infof("dispatching to backends: %# v", pretty.Formatter(marathonEvent))
		for _, backendImplementation := range backendconfig.RegisteredBackends {
			glog.Infof("dispatching event to backend '%s'", backendImplementation.Name())
			go backendImplementation.HandleUpdate(marathonEvent)
		}
	case "app_terminated_event":
		var marathonEvent backend.AppTerminatedEvent
		ginCtx.Bind(&marathonEvent)

		glog.Infof("dispatching to backends: %# v", pretty.Formatter(marathonEvent))
		for _, backendImplementation := range backendconfig.RegisteredBackends {
			glog.Infof("dispatching event to backend '%s'", backendImplementation.Name())
			go backendImplementation.HandleDestroy(marathonEvent)
		}
	default:
		msg := fmt.Sprintf("event type '%s' is not dispatched to any backend", eventType)
		glog.Errorf(msg)
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
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
