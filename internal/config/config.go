package config

import (
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Field names (viper keys).
const (
	FieldLogLevel      = "log_level"
	FieldServerAddress = "server_address"
	FieldEnableHTTPS   = "enable_https"
	FieldTLSCert       = "tls_cert"
	FieldTLSKey        = "tls_key"
)

// Flag names.
const (
	FlagLogLevel      = "log-level"
	FlagServerAddress = "server-address"
	FlagEnableHTTPS   = "enable-https"
	FlagTLSCert       = "tls-cert"
	FlagTLSKey        = "tls-key"
)

// Environment variable names.
const (
	EnvLogLevel      = "LOG_LEVEL"
	EnvServerAddress = "SERVER_ADDRESS"
	EnvEnableHTTPS   = "ENABLE_HTTPS"
	EnvTLSCert       = "TLS_CERT"
	EnvTLSKey        = "TLS_KEY"
)

// Log level values.
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)

// Default values.
const (
	DefaultEnvPrefix = "GK"

	DefaultLogLevel      = LogLevelInfo
	DefaultServerAddress = "0.0.0.0:8080"
	DefaultEnableHTTPS   = false
)

// ServerConfig - server configuration.
type ServerConfig struct {
	LogLevel      string `mapstructure:"LOG_LEVEL"`
	ServerAddress string `mapstructure:"SERVER_ADDRESS"`
	EnableHTTPS   bool   `mapstructure:"ENABLE_HTTPS"`
	TLSCert       string `mapstructure:"TLS_CERT"`
	TLSKey        string `mapstructure:"TLS_KEY"`
}

// Load - reads configuration with priority: defaults → env → flags.
func Load(envPrefix string) (ServerConfig, error) {
	v := viper.New()

	v.SetDefault(FieldLogLevel, DefaultLogLevel)
	v.SetDefault(FieldServerAddress, DefaultServerAddress)
	v.SetDefault(FieldEnableHTTPS, DefaultEnableHTTPS)
	v.SetDefault(FieldTLSCert, "")
	v.SetDefault(FieldTLSKey, "")

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	p := pflag.NewFlagSet("server", pflag.ContinueOnError)
	flagLog := p.String(FlagLogLevel, "", "log level (debug, info, warn, error)")
	flagAddr := p.String(FlagServerAddress, "", "server address (host:port)")
	flagHTTPS := p.Bool(FlagEnableHTTPS, false, "enable HTTPS")
	flagCert := p.String(FlagTLSCert, "", "path to TLS certificate file")
	flagKey := p.String(FlagTLSKey, "", "path to TLS private key file")

	if len(os.Args) > 1 {
		if err := p.Parse(os.Args[1:]); err != nil {
			return ServerConfig{}, err
		}
	}

	if p.Changed(FlagLogLevel) {
		v.Set(FieldLogLevel, *flagLog)
	}
	if p.Changed(FlagServerAddress) {
		v.Set(FieldServerAddress, *flagAddr)
	}
	if p.Changed(FlagEnableHTTPS) {
		v.Set(FieldEnableHTTPS, *flagHTTPS)
	}
	if p.Changed(FlagTLSCert) {
		v.Set(FieldTLSCert, *flagCert)
	}
	if p.Changed(FlagTLSKey) {
		v.Set(FieldTLSKey, *flagKey)
	}

	cfg := ServerConfig{
		LogLevel:      v.GetString(FieldLogLevel),
		ServerAddress: v.GetString(FieldServerAddress),
		EnableHTTPS:   v.GetBool(FieldEnableHTTPS),
		TLSCert:       v.GetString(FieldTLSCert),
		TLSKey:        v.GetString(FieldTLSKey),
	}

	return cfg, validate(cfg)
}
