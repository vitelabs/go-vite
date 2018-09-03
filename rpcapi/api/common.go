package api

type CommonApis interface {
	// if it  exists a log dir it will return it else return empty string
	LogDir() string
}
