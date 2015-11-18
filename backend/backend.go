package backend

type Backend interface {
	Register() (error, Backend) // this is for initializing stuff, establishing connections etc.
	HandleEvent(interface{})
}
