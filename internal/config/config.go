package config

import (
	"os"
	"strings"
	"time"

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
	FieldDatabaseURI   = "database_uri"
	FieldJWTSecret     = "jwt_secret"
	FieldJWTAccessTTL  = "jwt_access_ttl"
	FieldJWTRefreshTTL = "jwt_refresh_ttl"
	FieldEnableVault   = "enable_vault"
	FieldVaultAddress  = "vault_address"
	FieldVaultToken    = "vault_token"
	FieldVaultKeyPath  = "vault_key_path"
	FieldEncryptionKey = "encryption_key"
)

// Flag names.
const (
	FlagLogLevel      = "log-level"
	FlagServerAddress = "server-address"
	FlagEnableHTTPS   = "enable-https"
	FlagTLSCert       = "tls-cert"
	FlagTLSKey        = "tls-key"
	FlagDatabaseURI   = "database-uri"
	FlagJWTSecret     = "jwt-secret"
	FlagJWTAccessTTL  = "jwt-access-ttl"
	FlagJWTRefreshTTL = "jwt-refresh-ttl"
	FlagVault         = "vault"
	FlagVaultAddress  = "vault-address"
	FlagVaultToken    = "vault-token"
	FlagVaultKeyPath  = "vault-key-path"
	FlagEncryptionKey = "encryption-key"
)

// Environment variable names.
const (
	EnvLogLevel      = "LOG_LEVEL"
	EnvServerAddress = "SERVER_ADDRESS"
	EnvEnableHTTPS   = "ENABLE_HTTPS"
	EnvTLSCert       = "TLS_CERT"
	EnvTLSKey        = "TLS_KEY"
	EnvDatabaseURI   = "DATABASE_URI"
	EnvJWTSecret     = "JWT_SECRET"
	EnvJWTAccessTTL  = "JWT_ACCESS_TTL"
	EnvJWTRefreshTTL = "JWT_REFRESH_TTL"
	EnvEnableVault   = "ENABLE_VAULT"
	EnvVaultAddress  = "VAULT_ADDRESS"
	EnvVaultToken    = "VAULT_TOKEN"
	EnvVaultKeyPath  = "VAULT_KEY_PATH"
	EnvEncryptionKey = "ENCRYPTION_KEY"
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

	DefaultLogLevel         = LogLevelInfo
	DefaultServerAddress    = "0.0.0.0:8080"
	DefaultEnableHTTPS      = false
	DefaultDatabaseURI      = ""
	DefaultJWTAccessTTL     = 15 * time.Minute
	DefaultJWTAccessTTLStr  = "15m"
	DefaultJWTRefreshTTL    = 24 * time.Hour
	DefaultJWTRefreshTTLStr = "24h"
	DefaultEnableVault      = false
	DefaultVaultKeyPath     = "secret/data/gophkeeper/encryption"
)

// ServerConfig - server configuration.
// nolint: gosec
type ServerConfig struct {
	LogLevel      string        `mapstructure:"LOG_LEVEL"`
	ServerAddress string        `mapstructure:"SERVER_ADDRESS"`
	EnableHTTPS   bool          `mapstructure:"ENABLE_HTTPS"`
	TLSCert       string        `mapstructure:"TLS_CERT"`
	TLSKey        string        `mapstructure:"TLS_KEY"`
	DatabaseURI   string        `mapstructure:"DATABASE_URI"`
	JWTSecret     string        `mapstructure:"JWT_SECRET"`
	JWTAccessTTL  time.Duration `mapstructure:"JWT_ACCESS_TTL"`
	JWTRefreshTTL time.Duration `mapstructure:"JWT_REFRESH_TTL"`
	EnableVault   bool          `mapstructure:"ENABLE_VAULT"`
	VaultAddress  string        `mapstructure:"VAULT_ADDRESS"`
	VaultToken    string        `mapstructure:"VAULT_TOKEN"`
	VaultKeyPath  string        `mapstructure:"VAULT_KEY_PATH"`
	EncryptionKey string        `mapstructure:"ENCRYPTION_KEY"`
}

// Load - reads configuration with priority: defaults → env → flags.
func Load(envPrefix string) (ServerConfig, error) {
	v := viper.New()

	v.SetDefault(FieldLogLevel, DefaultLogLevel)
	v.SetDefault(FieldServerAddress, DefaultServerAddress)
	v.SetDefault(FieldEnableHTTPS, DefaultEnableHTTPS)
	v.SetDefault(FieldTLSCert, "")
	v.SetDefault(FieldTLSKey, "")
	v.SetDefault(FieldDatabaseURI, DefaultDatabaseURI)
	v.SetDefault(FieldJWTSecret, "")
	v.SetDefault(FieldJWTAccessTTL, DefaultJWTAccessTTLStr)
	v.SetDefault(FieldJWTRefreshTTL, DefaultJWTRefreshTTLStr)
	v.SetDefault(FieldEnableVault, DefaultEnableVault)
	v.SetDefault(FieldVaultAddress, "")
	v.SetDefault(FieldVaultToken, "")
	v.SetDefault(FieldVaultKeyPath, DefaultVaultKeyPath)
	v.SetDefault(FieldEncryptionKey, "")

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	p := pflag.NewFlagSet("server", pflag.ContinueOnError)
	flagLog := p.String(FlagLogLevel, DefaultLogLevel, "log level (debug, info, warn, error)")
	flagAddr := p.String(FlagServerAddress, DefaultServerAddress, "server address (host:port)")
	flagHTTPS := p.Bool(FlagEnableHTTPS, DefaultEnableHTTPS, "enable HTTPS")
	flagCert := p.String(FlagTLSCert, "", "path to TLS certificate file")
	flagKey := p.String(FlagTLSKey, "", "path to TLS private key file")
	flagDBURI := p.String(FlagDatabaseURI, DefaultDatabaseURI, "database connection URI")
	flagJWTSecret := p.String(FlagJWTSecret, "", "JWT secret key")
	flagJWTAccessTTL := p.Duration(FlagJWTAccessTTL, DefaultJWTAccessTTL, "JWT access token TTL")
	flagJWTRefreshTTL := p.Duration(FlagJWTRefreshTTL, DefaultJWTRefreshTTL, "JWT refresh token TTL")
	flagVault := p.Bool(FlagVault, DefaultEnableVault, "enable Vault for encryption key storage")
	flagVaultAddress := p.String(FlagVaultAddress, "", "Vault address (e.g. http://127.0.0.1:8200)")
	flagVaultToken := p.String(FlagVaultToken, "", "Vault token")
	flagVaultKeyPath := p.String(FlagVaultKeyPath, DefaultVaultKeyPath, "Vault KV v2 key path")
	flagEncryptionKey := p.String(FlagEncryptionKey, "", "base64-encoded encryption key (fallback without Vault)")

	if len(os.Args) > 1 {
		err := p.Parse(os.Args[1:])
		if err != nil {
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
	if p.Changed(FlagDatabaseURI) {
		v.Set(FieldDatabaseURI, *flagDBURI)
	}
	if p.Changed(FlagJWTSecret) {
		v.Set(FieldJWTSecret, *flagJWTSecret)
	}
	if p.Changed(FlagJWTAccessTTL) {
		v.Set(FieldJWTAccessTTL, *flagJWTAccessTTL)
	}
	if p.Changed(FlagJWTRefreshTTL) {
		v.Set(FieldJWTRefreshTTL, *flagJWTRefreshTTL)
	}
	if p.Changed(FlagVault) {
		v.Set(FieldEnableVault, *flagVault)
	}
	if p.Changed(FlagVaultAddress) {
		v.Set(FieldVaultAddress, *flagVaultAddress)
	}
	if p.Changed(FlagVaultToken) {
		v.Set(FieldVaultToken, *flagVaultToken)
	}
	if p.Changed(FlagVaultKeyPath) {
		v.Set(FieldVaultKeyPath, *flagVaultKeyPath)
	}
	if p.Changed(FlagEncryptionKey) {
		v.Set(FieldEncryptionKey, *flagEncryptionKey)
	}

	cfg := ServerConfig{
		LogLevel:      v.GetString(FieldLogLevel),
		ServerAddress: v.GetString(FieldServerAddress),
		EnableHTTPS:   v.GetBool(FieldEnableHTTPS),
		TLSCert:       v.GetString(FieldTLSCert),
		TLSKey:        v.GetString(FieldTLSKey),
		DatabaseURI:   v.GetString(FieldDatabaseURI),
		JWTSecret:     v.GetString(FieldJWTSecret),
		JWTAccessTTL:  v.GetDuration(FieldJWTAccessTTL),
		JWTRefreshTTL: v.GetDuration(FieldJWTRefreshTTL),
		EnableVault:   v.GetBool(FieldEnableVault),
		VaultAddress:  v.GetString(FieldVaultAddress),
		VaultToken:    v.GetString(FieldVaultToken),
		VaultKeyPath:  v.GetString(FieldVaultKeyPath),
		EncryptionKey: v.GetString(FieldEncryptionKey),
	}

	return cfg, validate(cfg)
}
