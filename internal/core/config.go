package core

import (
	"log/slog"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/starbx/brew-api/internal/core/srv"
)

func MustLoadBaseConfig(path string) CoreConfig {
	if path == "" {
		return LoadBaseConfigFromENV()
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var conf CoreConfig
	if err = toml.Unmarshal(raw, &conf); err != nil {
		panic(err)
	}
	return conf
}

func LoadBaseConfigFromENV() CoreConfig {
	var c CoreConfig
	c.FromENV()
	return c
}

type CoreConfig struct {
	Addr     string   `toml:"addr"`
	Log      Log      `toml:"log"`
	Postgres PGConfig `toml:"postgres"`

	AI srv.AIConfig `toml:"ai"`

	Security Security `toml:"security"`

	Prompt Prompt `toml:"prompt"`
}

type Prompt struct {
	Base         string `toml:"base"`
	Query        string `toml:"query"`
	ChatSummary  string `toml:"chat_summary"`
	EnhanceQuery string `toml:"enhance_query"`
	SessionName  string `toml:"session_name"`
}

type Security struct {
	PublicKey string `json:"public_key"`
}

func (c *CoreConfig) FromENV() {
	c.Addr = os.Getenv("BREW_API_SERVICE_ADDRESS")
	c.Log.FromENV()
	c.Postgres.FromENV()
	c.AI.FromENV()
}

type PGConfig struct {
	DSN string `toml:"dsn"`
}

func (m *PGConfig) FromENV() {
	m.DSN = os.Getenv("BREW_API_POSTGRESQL_DSN")
}

func (c PGConfig) FormatDSN() string {
	return c.DSN
}

type Log struct {
	Level string `toml:"level"`
	Path  string `toml:"path"`
}

func (l *Log) FromENV() {
	l.Level = os.Getenv("BREW_API_LOG_LEVEL")
	l.Path = os.Getenv("BREW_API_LOG_PATH")
}

func (l *Log) SlogLevel() slog.Level {
	switch strings.ToLower(l.Level) {
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
