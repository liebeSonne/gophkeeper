package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

var errInvalidLogLevel = errors.New("invalid log level")
var errInvalidServerAddress = errors.New("invalid server address")
var errInvalidServerAddressHost = errors.New("invalid server address host")
var errInvalidServerAddressPort = errors.New("invalid server address port (must be 1-65535)")
var errEmptyTLSCert = errors.New("empty TLS certificate")
var errEmptyTLSKey = errors.New("empty TLS key")
var errEmptyJWTSecret = errors.New("empty JWT secret")

var validLogLevels = map[string]bool{
	LogLevelDebug: true,
	LogLevelInfo:  true,
	LogLevelWarn:  true,
	LogLevelError: true,
	LogLevelFatal: true,
}

func validate(cfg ServerConfig) error {
	if !validLogLevels[cfg.LogLevel] {
		return fmt.Errorf("%w: %s", errInvalidLogLevel, cfg.LogLevel)
	}

	host, port, err := net.SplitHostPort(cfg.ServerAddress)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidServerAddress, cfg.ServerAddress)
	}

	if host != "" && host != "0.0.0.0" {
		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("%w: %s", errInvalidServerAddressHost, host)
		}
	}

	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("%w: %s", errInvalidServerAddressPort, port)
	}

	if cfg.EnableHTTPS {
		if cfg.TLSCert == "" {
			return errEmptyTLSCert
		}
		if cfg.TLSKey == "" {
			return errEmptyTLSKey
		}
	}

	if cfg.JWTSecret == "" {
		return errEmptyJWTSecret
	}

	return nil
}
