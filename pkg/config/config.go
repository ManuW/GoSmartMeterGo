package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// SerialConfig defines settings for the optical SML probe.
type SerialConfig struct {
	Port     string `mapstructure:"port"`
	BaudRate int    `mapstructure:"baud_rate"`
	DataBits int    `mapstructure:"data_bits"`
	Parity   string `mapstructure:"parity"`
	StopBits int    `mapstructure:"stop_bits"`
	RTS      bool   `mapstructure:"rts"`
}

// SpeedwireConfig defines settings for SMA Speedwire listener.
type SpeedwireConfig struct {
	MulticastGroup string `mapstructure:"multicast_group"`
	Port           int    `mapstructure:"port"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

// DatabaseConfig defines SQLite connection settings.
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// TelegramConfig defines Telegram Bot notifications settings.
type TelegramConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	BotToken string   `mapstructure:"bot_token"`
	ChatIDs  []string `mapstructure:"chat_ids"`
}

// RsyncConfig defines settings for SSH/rsync transfers.
type RsyncConfig struct {
	Host       string `mapstructure:"host"`
	Username   string `mapstructure:"username"`
	RemotePath string `mapstructure:"remote_path"`
	SSHKeyPath string `mapstructure:"ssh_key_path"`
}

// BackupConfig defines settings for database backup and replication.
type BackupConfig struct {
	Enabled       bool        `mapstructure:"enabled"`
	Provider      string      `mapstructure:"provider"`
	IntervalHours int         `mapstructure:"interval_hours"`
	KeepRevisions int         `mapstructure:"keep_revisions"`
	Rsync         RsyncConfig `mapstructure:"rsync"`
}

// WebConfig defines the embedded dashboard server settings.
type WebConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// Config is the top-level configuration structure.
type Config struct {
	Serial    SerialConfig    `mapstructure:"serial"`
	Speedwire SpeedwireConfig `mapstructure:"speedwire"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Telegram  TelegramConfig  `mapstructure:"telegram"`
	Web       WebConfig       `mapstructure:"web"`
	Backup    BackupConfig    `mapstructure:"backup"`
}

// LoadConfig loads the configuration from a file or environment variables.
func LoadConfig(cfgFile string) (*Config, error) {
	v := viper.New()

	// Set Defaults
	v.SetDefault("serial.port", "/dev/ttyAMA0")
	v.SetDefault("serial.baud_rate", 9600)
	v.SetDefault("serial.data_bits", 8)
	v.SetDefault("serial.parity", "none")
	v.SetDefault("serial.stop_bits", 1)
	v.SetDefault("serial.rts", true)

	v.SetDefault("speedwire.multicast_group", "239.12.255.254")
	v.SetDefault("speedwire.port", 9522)
	v.SetDefault("speedwire.timeout_seconds", 120)

	v.SetDefault("database.path", "smartmeter.db")

	v.SetDefault("telegram.enabled", false)
	v.SetDefault("telegram.bot_token", "")
	v.SetDefault("telegram.chat_ids", []string{})

	v.SetDefault("web.host", "0.0.0.0")
	v.SetDefault("web.port", 8080)

	v.SetDefault("backup.enabled", false)
	v.SetDefault("backup.provider", "rsync")
	v.SetDefault("backup.interval_hours", 24)
	v.SetDefault("backup.keep_revisions", 10)
	v.SetDefault("backup.rsync.host", "")
	v.SetDefault("backup.rsync.username", "")
	v.SetDefault("backup.rsync.remote_path", "")
	v.SetDefault("backup.rsync.ssh_key_path", "")

	// Environment variables setup
	v.SetEnvPrefix("SMARTMETER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file if specified, or look for default config.yaml
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.config/gosmartmeter")
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// If not found, we proceed since env vars or defaults might be sufficient
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &cfg, nil
}
