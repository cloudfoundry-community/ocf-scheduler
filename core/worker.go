package core

type WorkerService interface {
	Submit(func())
	StopWait()
}
