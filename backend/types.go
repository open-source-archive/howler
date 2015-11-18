package backend

import (
	"time"
)

type Event struct {
	Eventtype string    `json:"eventType"`
	Timestamp time.Time `json:"timestamp"`
}

type ApiRequestEvent struct {
	Event
	Clientip      string `json:"clientIp"`
	URI           string `json:"uri"`
	Appdefinition struct {
		Args            []interface{} `json:"args"`
		Backofffactor   float64       `json:"backoffFactor"`
		Backoffseconds  int           `json:"backoffSeconds"`
		Cmd             string        `json:"cmd"`
		Constraints     []interface{} `json:"constraints"`
		Container       interface{}   `json:"container"`
		Cpus            float64       `json:"cpus"`
		Dependencies    []interface{} `json:"dependencies"`
		Disk            float64       `json:"disk"`
		Env             struct{}      `json:"env"`
		Executor        string        `json:"executor"`
		Healthchecks    []interface{} `json:"healthChecks"`
		ID              string        `json:"id"`
		Instances       int           `json:"instances"`
		Ports           []int         `json:"ports"`
		Requireports    bool          `json:"requirePorts"`
		Storeurls       []interface{} `json:"storeUrls"`
		Upgradestrategy struct {
			Minimumhealthcapacity float64 `json:"minimumHealthCapacity"`
		} `json:"upgradeStrategy"`
		Uris    []interface{} `json:"uris"`
		User    interface{}   `json:"user"`
		Version time.Time     `json:"version"`
	} `json:"appDefinition"`
}

type StatusUpdateEvent struct {
	Event
	Slaveid    string    `json:"slaveId"`
	Taskid     string    `json:"taskId"`
	Taskstatus string    `json:"taskStatus"`
	Appid      string    `json:"appId"`
	Host       string    `json:"host"`
	Ports      []int     `json:"ports"`
	Version    time.Time `json:"version"`
}
