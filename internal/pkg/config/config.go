package config

type Config struct {
	Port int    `default:"3000"`
	Host string `default:"localhost"`
	LogLevel string `default:"WARN"`
}
