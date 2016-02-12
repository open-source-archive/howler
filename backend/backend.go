package backend

type Backend interface {
	Name() string
	Register() error // this is for initializing stuff, establishing connections etc.
	HandleCreate(ApiRequestEvent)
	HandleUpdate(StatusUpdateEvent)
	HandleDestroy(AppTerminatedEvent)
}
