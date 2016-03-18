package backend

//Backend provides general methods
type Backend interface {
	Name() string
	Register() error // this is for initializing stuff, establishing connections etc.
	HandleCreate(APIRequestEvent)
	HandleUpdate(StatusUpdateEvent)
	HandleDestroy(AppTerminatedEvent)
}
