package data

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

type AppConfig struct {
	Name    string `default:"arktie"`
	Version string `default:"0.0.0"`
	Debug   bool   `mapstructure:"debug" env:"APP_DEBUG"`

	URL      *url.URL `mapstructure:"-"`
	URLValue string   `mapstructure:"url" env:"APP_URL"`

	Key      []byte `mapstructure:"-"`
	KeyValue string `mapstructure:"key" env:"APP_KEY"`

	SigningPrivateKey      ed25519.PrivateKey `mapstructure:"-"`
	SigningPrivateKeyValue string             `mapstructure:"sign_private_key" env:"SIGN_PRIVATE_KEY"`
	SigningPublicKey       ed25519.PublicKey  `mapstructure:""`
	SigningPublicKeyValue  string             `mapstructure:"sign_public_key" env:"SIGN_PUBLIC_KEY"`
}

func (c AppConfig) IsLocal() bool {
	if c.URL == nil {
		return false
	}

	switch c.URL.Hostname() {
	case "localhost", "127.0.0.1", "::1", "":
		return true
	}

	return false
}

type ServerConfig struct {
	Addr string `mapstructure:"addr" env:"SERVER_ADDR" default:"8000"`
}

type SQLConfig struct {
	Driver string `mapstructure:"driver" env:"SQL_DRIVER" default:"sqlite"`
	DSN    string `mapstructure:"dsn" env:"SQL_DSN" default:"file:./arktie.sqlite?_pragma=foreign_keys(1)"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr" env:"REDIS_ADDR"`
	Password string `mapstructure:"pass" env:"REDIS_PASS"`
}

type DatabaseConfig struct {
	SQL   SQLConfig   `mapstructure:"sql" default:""`
	Redis RedisConfig `mapstructure:"redis" default:""`
}

type FirehoseConfig struct {
	RelayURL      *url.URL `mapstructure:"-"`
	RelayURLValue string   `mapstructure:"relay_url" env:"FIREHOSE_RELAY_URL" default:"wss://relay1.us-east.bsky.network"`
}

type ServiceConfig struct {
	Database DatabaseConfig `mapstructure:"database" default:""`
	Firehose FirehoseConfig `mapstructure:"firehose" default:""`
}

type Config struct {
	App     AppConfig     `mapstructure:"app" default:""`
	Server  ServerConfig  `mapstructure:"server" default:""`
	Service ServiceConfig `mapstructure:"service" default:""`
}

func NewConfig(name, version string, paths ...string) (*Config, error) {
	config.WithOptions(config.ParseEnv, config.ParseTime, config.ParseDefault)
	config.AddDriver(yaml.Driver)

	cfg := &Config{}

	var cfgPaths []string
	for _, p := range paths {
		stat, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		if !stat.IsDir() {
			cfgPaths = append(cfgPaths, p)
			continue
		}

		filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			ext := filepath.Ext(path)
			if !d.IsDir() && (ext == ".yaml" || ext == ".yml") {
				cfgPaths = append(cfgPaths, path)
			}

			return nil
		})
	}

	if err := config.LoadFiles(cfgPaths...); err != nil {
		return nil, err
	}

	if err := config.BindStruct("", cfg); err != nil {
		return nil, err
	}

	if name != "" {
		cfg.App.Name = name
	}
	if version != "" {
		cfg.App.Version = version
	}

	if key, err := parseKey(cfg.App.KeyValue); err != nil {
		return nil, err
	} else {
		cfg.App.Key = key
	}

	if pub, priv, err := parseSignKey(cfg.App.SigningPublicKeyValue, cfg.App.SigningPrivateKeyValue); err != nil {
		return nil, err
	} else {
		cfg.App.SigningPublicKey = pub
		cfg.App.SigningPrivateKey = priv
	}

	if u, err := url.Parse(cfg.App.URLValue); err != nil {
		return nil, err
	} else {
		cfg.App.URL = u
	}
	if u, err := url.Parse(cfg.Service.Firehose.RelayURLValue); err != nil {
		return nil, err
	} else {
		cfg.Service.Firehose.RelayURL = u
	}

	return cfg, nil
}

func parseKey(value string) (ret []byte, err error) {
	if len(value) == 0 {
		return nil, errors.New("missing app.key")
	}

	switch {
	case strings.HasPrefix(value, "hex:"):
		trimmed := strings.TrimPrefix(value, "hex:")
		return hex.DecodeString(trimmed)
	case strings.HasPrefix(value, "base64:"):
		trimmed := strings.TrimPrefix(value, "base64:")
		return base64.StdEncoding.DecodeString(trimmed)
	}

	return []byte(value), nil
}

func parseSignKey(pubv, priv string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pubBlock, _ := pem.Decode([]byte(pubv))
	if pubBlock == nil {
		return nil, nil, errors.New("failed to decode public key PEM")
	}

	pubParsed, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse public key: %w", err)
	}

	pubkey, ok := pubParsed.(ed25519.PublicKey)
	if !ok {
		return nil, nil, errors.New("public key is not ed25519")
	}

	privBlock, _ := pem.Decode([]byte(priv))
	if privBlock == nil {
		return nil, nil, errors.New("failed to decode private key PEM")
	}

	privParsed, err := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse private key: %w", err)
	}

	privkey, ok := privParsed.(ed25519.PrivateKey)
	if !ok {
		return nil, nil, errors.New("private key is not ed25519")
	}

	msg := []byte("validation-test")
	if !ed25519.Verify(pubkey, msg, ed25519.Sign(privkey, msg)) {
		return nil, nil, errors.New("signing key validation error")
	}

	return pubkey, privkey, nil
}
