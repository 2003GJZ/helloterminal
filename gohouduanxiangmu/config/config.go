package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Server struct {
		Port int    `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server"`

	Database struct {
		Type string `mapstructure:"type"`
		Path string `mapstructure:"path"`
	} `mapstructure:"database"`

	Session struct {
		Timeout         int `mapstructure:"timeout"`
		CleanupInterval int `mapstructure:"cleanup_interval"`
	} `mapstructure:"session"`

	Security struct {
		JWTSecret   string `mapstructure:"jwt_secret"`
		TokenExpiry int    `mapstructure:"token_expiry"`
	} `mapstructure:"security"`

	SSH struct {
		Timeout    int `mapstructure:"timeout"`
		Keepalive  int `mapstructure:"keepalive"`
		BufferSize int `mapstructure:"buffer_size"`
	} `mapstructure:"ssh"`

	Log struct {
		Level      string `mapstructure:"level"`
		File       string `mapstructure:"file"`
		MaxSize    int    `mapstructure:"max_size"`
		MaxBackups int    `mapstructure:"max_backups"`
		MaxAge     int    `mapstructure:"max_age"`
		Compress   bool   `mapstructure:"compress"`
	} `mapstructure:"log"`
}

var (
	config *Config
	once   sync.Once
)

// LoadConfig 加载配置文件
func LoadConfig() *Config {
	once.Do(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./config")
		viper.AddConfigPath(".")

		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Error reading config file: %s", err)
		}

		config = &Config{}
		if err := viper.Unmarshal(config); err != nil {
			log.Fatalf("Unable to decode into config struct: %s", err)
		}
	})

	return config
}

// GetConfig 获取配置实例
func GetConfig() *Config {
	if config == nil {
		return LoadConfig()
	}
	return config
}
