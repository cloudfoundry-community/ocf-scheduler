package core

type LogService interface {
	Trace(string, string)
	Debug(string, string)
	Info(string, string)
	Warn(string, string)
	Error(string, string)
	Fatal(string, string)
	Panic(string, string)
}
