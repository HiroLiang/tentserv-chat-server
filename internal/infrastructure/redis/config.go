package redis

type ClientConfig struct {
	Addr     string
	Password string
	DB       int
}
