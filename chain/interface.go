package chain

type Net interface {
	Broadcast() error
}
