package backend

import (
	"github.com/golang/glog"
)

type DummyBackend struct {
	name string
}

func (be *DummyBackend) Name() string {
	return "DummyBackend"
}

func (be *DummyBackend) Register() error {
	return nil
}

func (be *DummyBackend) HandleUpdate(e StatusUpdateEvent) {
	glog.Infof("%+v\n", e)
}

func (be *DummyBackend) HandleCreate(e ApiRequestEvent)     {}
func (be *DummyBackend) HandleDestroy(e AppTerminatedEvent) {}
