package backend

type Backend interface {
	Name() string
	Register() (error, Backend) // this is for initializing stuff, establishing connections etc.
	HandleEvent(interface{})
}
