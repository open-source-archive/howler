package backend

import (
	"github.com/golang/glog"
	"github.com/kr/pretty"
)

type Zmon2 struct {
	name string
}

func (z Zmon2) Register() (error, Backend) {
	return nil, Zmon2{"zmon2"}
}

func (z Zmon2) HandleEvent(event interface{}) {
	event, ok := event.(StatusUpdateEvent)
	if !ok {
		glog.Errorf("Backend %s: unable to handle received event type", z.name)
		return
	}
	glog.Infof("Backend %s: handling event: %# v", z.name, pretty.Formatter(event))
}
