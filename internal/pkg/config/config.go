package config

type Config struct {
	Port int    `default:"5555"`
	Host string `default:"localhost"`
	LogLevel string `default:"DEBUG"`
	CertPath string `default:"certs"`
}
