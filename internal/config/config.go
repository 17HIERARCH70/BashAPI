package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
)

type Config struct {
	Env      string         `yaml:"env" env-default:"local"`
	Server   ServerConfig   `yaml:"server"`
	Postgres PostgresConfig `yaml:"postgres"`
	Commands CommandsConfig `yaml:"commands"`
}
type ServerConfig struct {
	Host         string `yaml:"host" env-default:"localhost"`
	Port         int    `yaml:"port" env-default:"8080"`
	ReadTimeout  int    `yaml:"read_timeout" env-default:"10"`
	WriteTimeout int    `yaml:"write_timeout" env-default:"10"`
}
type PostgresConfig struct {
	Host     string `yaml:"host" env-default:"localhost"`
	Port     int    `yaml:"port" env-default:"5432"`
	User     string `yaml:"user" env-default:"postgres"`
	Password string `yaml:"password"`
	Database string `yaml:"database" env-default:"SocialManagerDB"`
	SSLMode  string `yaml:"ssl_mode" env-default:"disable"`
}

type CommandsConfig struct {
	MaxConcurrent int `yaml:"max_concurrent" env-default:"100"`
	Timeout       int `yaml:"timeout" env-default:"100"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config is empty")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("config path if empty " + err.Error())
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
