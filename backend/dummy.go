package backend

import (
	"github.com/golang/glog"
)

type DummyBackend struct {
	name string
}

func (be DummyBackend) Name() string {
	return be.name
}

func (be DummyBackend) Register() (error, Backend) {
	return nil, DummyBackend{name: "DummyBackend"}
}

func (be DummyBackend) HandleEvent(event interface{}) {
	event, ok := event.(StatusUpdateEvent)
	if !ok {
		glog.Errorf("Backend %s: unable to handle received event type", be.name)
		return
	}
}
