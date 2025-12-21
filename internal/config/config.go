package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string           `yaml:"env"`
	StoragePath string           `yaml:"storage_path" env-required:"true"`
	HTTPServer  HTTPServerConfig `yaml:"http_server"`
	Migrations  MigrationsConfig `yaml:"migrations"`
	Clients     ClientsConfig    `yaml:"clients"`
	AppSecret   string           `yaml:"app_secret" env-required:"true"`
}

type HTTPServerConfig struct {
	Address         string        `yaml:"address" env-default:"localhost:8080"`
	Timeout         time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"10s"`
}

type Client struct {
	Address  string        `yaml:"addr" env-required:"true"`
	Timeout  time.Duration `yaml:"timeout" env-default:"5s"`
	Retries  int           `yaml:"retries" env-default:"3"`
	Insecure bool          `yaml:"insecure" env-default:"true"`
}

type ClientsConfig struct {
	SSO Client `yaml:"sso"`
}

type MigrationsConfig struct {
	MigrationsPath string `yaml:"migrations_path" env-default:"./migrations"`
	MigrationTable string `yaml:"migration_table" env-default:"migrations"`
}

func MustLoad() *Config {
	path := fetchConfigPath()
	if path == "" {
		panic("config file path is empty")
	}

	return MustLoadByPath(path)
}

func MustLoadByPath(configPath string) *Config {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file not found: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
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
