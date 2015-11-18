package backend

type Backend interface {
	register()
	handleEvent()
}
