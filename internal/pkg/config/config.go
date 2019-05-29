package config

type Config struct {
	Port int    `env:"PORT" envDefault:"3000"`
	Host string `env:"HOST" envDefault:"localhost"`
}
