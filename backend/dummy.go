package backend

import (
	"github.com/golang/glog"
)

// DummyBackend general fields
type DummyBackend struct {
	name string
}

// Name return Dummy backend name
func (be *DummyBackend) Name() string {
	return be.name
}

// Register initializes DummyBackend
func (be *DummyBackend) Register() error {
	be.name = "DummyBackend"
	return nil
}

//HandleUpdate reaps update events from Marathon
func (be *DummyBackend) HandleUpdate(e StatusUpdateEvent) {
	glog.Infof("%+v\n", e)
}

//HandleCreate reaps API request events from Marathon
func (be *DummyBackend) HandleCreate(e APIRequestEvent) {}

//HandleDestroy reaps API terminated events from Marathon
func (be *DummyBackend) HandleDestroy(e AppTerminatedEvent) {}
