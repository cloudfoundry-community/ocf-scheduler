package core

type LogService interface {
	Info(string, string)
	Error(string, string)
}
