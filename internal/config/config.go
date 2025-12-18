package config

import (
	"flag"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `yaml:"env"`
	StoragePath string `yaml:"storage_path" env-required:"true"`
	HTTPServer  HTTPServerConfig
	Migrations  MigrationsConfig
}

type HTTPServerConfig struct {
	Address     string `yaml:"address" env-default:":8080"`
	Timeout     int    `yaml:"timeout" env-default:"5"`
	IdleTimeout int    `yaml:"idle_timeout" env-default:"60"`
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
