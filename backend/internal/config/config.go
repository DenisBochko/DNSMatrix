package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

var ErrConfigPathIsEmpty = errors.New("config path is empty")

type Config struct {
	App        `yaml:"app"`
	Logger     `yaml:"log"`
	Database   `yaml:"database"`
	Redis      `yaml:"redis"`
	HTTPServer `yaml:"http_server"`
	Mailer     `yaml:"mailer"`
	Key        `yaml:"key"`
	Kafka      `yaml:"kafka"`
	Elastic    `yaml:"elastic"`
	Geo        `yaml:"geo"`
}

type App struct {
	ServiceName string `yaml:"service_name"`
	Version     string `yaml:"version"`
}

type Logger struct {
	Level      string   `yaml:"level"`
	FormatJSON bool     `yaml:"format_json"`
	Rotation   Rotation `yaml:"rotation"`
}

type Rotation struct {
	File       string `json:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

type Database struct {
	Host      string    `yaml:"host"`
	Port      uint16    `yaml:"port"`
	User      string    `yaml:"user"`
	Password  string    `yaml:"password"`
	Name      string    `yaml:"name"`
	SSLMode   string    `yaml:"ssl_mode"`
	MaxConns  int32     `yaml:"max_conns"`
	MinConns  int32     `yaml:"min_conns"`
	Migration Migration `yaml:"migration"`
}

type Migration struct {
	Path      string `yaml:"path"`
	AutoApply bool   `yaml:"auto_apply"`
}

type Redis struct {
	Enable   bool   `yaml:"enable"`
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type HTTPServer struct {
	Host     string  `yaml:"host"`
	Port     uint16  `yaml:"port"`
	BasePath string  `yaml:"base_path"`
	Timeout  Timeout `yaml:"timeout"`
	CORS     CORS    `yaml:"cors"`
	JWT      JWT     `yaml:"jwt"`
}

type Timeout struct {
	Request time.Duration `yaml:"request"`
	Read    time.Duration `yaml:"read"`
	Write   time.Duration `yaml:"write"`
	Idle    time.Duration `yaml:"idle"`
}

type CORS struct {
	Enabled          bool          `yaml:"enabled"`
	AllowAllOrigins  bool          `yaml:"allow_all_origins"`
	AllowOrigins     []string      `yaml:"allow_origins"`
	AllowMethods     []string      `yaml:"allow_methods"`
	AllowHeaders     []string      `yaml:"allow_headers"`
	ExposeHeaders    []string      `yaml:"expose_headers"`
	AllowCredentials bool          `yaml:"allow_credentials"`
	MaxAge           time.Duration `yaml:"max_age"`
	AllowWebSockets  bool          `yaml:"allow_websockets"`
	AllowFiles       bool          `yaml:"allow_files"`
}

type JWT struct {
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`
}

type Mailer struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	UseTLS   bool   `yaml:"use_tls"`
}

type Key struct {
	PublicKey  string `yaml:"public"`
	PrivateKey string `yaml:"private"`
}

type Elastic struct {
	Addresses []string      `yaml:"addresses"`
	Username  string        `yaml:"username"`
	Password  string        `yaml:"password"`
	CloudID   string        `yaml:"cloud_id"`
	APIKey    string        `yaml:"api_key"`
	Timeout   time.Duration `yaml:"timeout"`
}

type Kafka struct {
	Brokers    []string   `yaml:"brokers"`
	Subscriber Subscriber `yaml:"subscriber"`
	Producer   Producer   `yaml:"producer"`
}

type Subscriber struct {
	Name        string `yaml:"name"`
	WorkerCount int    `yaml:"worker_count"`
	Topic       string `yaml:"topic"`
	GroupID     string `yaml:"group_id"`
}

type Producer struct {
	Name         string        `yaml:"name"`
	WorkerCount  int           `yaml:"worker_count"`
	PollInterval time.Duration `yaml:"poll_interval"`
	BatchSize    int           `yaml:"batch_size"`
}

type Geo struct {
	GeoLiteCountryPath string `yaml:"geo_lite_country_path"`
	GeoLiteASNPath     string `yaml:"geo_lite_asn_path"`
}

func MustLoadConfig() *Config {
	cfg, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	return cfg
}

func LoadConfig() (*Config, error) {
	path := fetchConfigPath()
	if path == "" {
		return nil, ErrConfigPathIsEmpty
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", path)
	}

	var config Config

	if err := cleanenv.ReadConfig(path, &config); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &config, nil
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
