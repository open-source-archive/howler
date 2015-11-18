package backend

import  (

	"github.com/golang/glog"
)

type Zmon2 struct {
    name string
}

func (z Zmon2) Register() (error, Backend) {
    return nil, Zmon2{"zmon2"}
}

func (z Zmon2) HandleEvent(event Event) {
    glog.Infof("Backend %s: handling %v\n", z.name, event)
}
