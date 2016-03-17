package backend

//Event provides abbasic type containing only the fields all Marathon events have in common.
type Event struct {
	Eventtype string `json:"eventType"`
	Timestamp string `json:"timestamp"`
}

// All following event types are generated with https://mholt.github.io/json-to-go/
// from Marathon Event Bus docu examples
// (https://raw.githubusercontent.com/mesosphere/marathon/master/docs/docs/event-bus.md).

//APIRequestEvent for requests that modifies an app (create, update, delete)
type APIRequestEvent struct {
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
		Version string        `json:"version"`
	} `json:"appDefinition"`
}

//StatusUpdateEvent for status of a task changes
type StatusUpdateEvent struct {
	Event
	Slaveid    string `json:"slaveId"`
	Taskid     string `json:"taskId"`
	Taskstatus string `json:"taskStatus"`
	Appid      string `json:"appId"`
	Host       string `json:"host"`
	Ports      []int  `json:"ports"`
	Version    string `json:"version"`
}

//AppTerminatedEvent for requests that deletes an app
type AppTerminatedEvent struct {
	Event
	Appid string `json:"appId"`
}
