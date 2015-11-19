package backend

import (
	"github.com/golang/glog"
	"github.com/kr/pretty"
)

const name = "dummy"

type DummyBackend struct {
	name string
}

func (be DummyBackend) Register() (error, Backend) {
	return nil, DummyBackend{name: name}
}

func (be DummyBackend) HandleEvent(event interface{}) {
	event, ok := event.(StatusUpdateEvent)
	if !ok {
		glog.Errorf("Backend %s: unable to handle received event type", be.name)
		return
	}
	glog.Infof("Backend %s: handling event: %# v", be.name, pretty.Formatter(event))
}
