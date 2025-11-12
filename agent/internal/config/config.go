package config

import (
	"errors"
	"flag"
	"os"

	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

var ErrConfigPathIsEmpty = errors.New("config path is empty")

type Config struct {
	App        `yaml:"app"`
	Subscriber `yaml:"subscriber"`
	Publisher  `yaml:"publisher"`
}

type App struct {
	AgentID uuid.UUID `yaml:"agent_id" env:"APP_AGENT_ID"`
	Region  string    `yaml:"region" env:"APP_REGION"`
}

type Subscriber struct {
	Brokers    []string `yaml:"brokers" env:"SUBSCRIBER_BROKERS" env-separator:","`
	GroupID    string   `yaml:"group_id" env:"SUBSCRIBER_GROUP_ID"`
	Topic      string   `yaml:"topic" env:"SUBSCRIBER_TOPIC"`
	BufferSize int      `yaml:"buffer_size" env:"SUBSCRIBER_BUFFER_SIZE"`
}

type Publisher struct {
	Brokers []string `yaml:"brokers" env:"PUBLISHER_BROKERS" env-separator:","`
	Topic   string   `yaml:"topic" env:"PUBLISHER_TOPIC"`
}

func MustLoadConfig() *Config {
	cfg, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	return cfg
}

func LoadConfig() (*Config, error) {
	var cfg Config
	path := fetchConfigPath()

	if path != "" {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, err
		}

		return &cfg, nil
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func MustPrintConfig(cfg *Config) {
	if err := PrintConfig(cfg); err != nil {
		panic(err)
	}
}

func PrintConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	println(string(data))

	return nil
}

func fetchConfigPath() string {
	var result string

	flag.StringVar(&result, "config", "", "Path to config file")
	flag.Parse()

	if result == "" {
		result = os.Getenv("CONFIG_PATH")
	}

	return result
}
