package storage

// Config defines the top-level configuration for pluggable storage backends.
type Config struct {
	Mode       string         `yaml:"mode" json:"mode"` // "local", "team", "enterprise"
	SQLite     SQLiteConfig   `yaml:"sqlite" json:"sqlite"`
	Postgres   PostgresConfig `yaml:"postgres" json:"postgres"`
	Qdrant     QdrantConfig   `yaml:"qdrant" json:"qdrant"`
}

// SQLiteConfig contains configuration options for Local Mode (SQLite + sqlite-vec).
type SQLiteConfig struct {
	Path        string `yaml:"path" json:"path"`                 // File path e.g. ~/.zuri/zuri.db
	BusyTimeout int    `yaml:"busy_timeout" json:"busy_timeout"` // Milliseconds
	PragmaWAL   bool   `yaml:"pragma_wal" json:"pragma_wal"`
}

// PostgresConfig contains configuration options for Team Mode (PostgreSQL + pgvector).
type PostgresConfig struct {
	Host           string `yaml:"host" json:"host"`
	Port           int    `yaml:"port" json:"port"`
	User           string `yaml:"user" json:"user"`
	Password       string `yaml:"password" json:"password"`
	DBName         string `yaml:"dbname" json:"dbname"`
	SSLMode        string `yaml:"sslmode" json:"sslmode"`
	MaxConnections int    `yaml:"max_connections" json:"max_connections"`
}

// QdrantConfig contains configuration options for Enterprise Mode (Qdrant Cluster).
type QdrantConfig struct {
	Endpoint   string `yaml:"endpoint" json:"endpoint"` // e.g. http://127.0.0.1:6333
	APIKeyEnv  string `yaml:"api_key_env" json:"api_key_env"`
	Collection string `yaml:"collection" json:"collection"`
	TimeoutSec int    `yaml:"timeout_sec" json:"timeout_sec"`
}

// DefaultConfig returns sensible defaults for zero-configuration Local Mode.
func DefaultConfig() Config {
	return Config{
		Mode: "local",
		SQLite: SQLiteConfig{
			Path:        "~/.zuri/zuri.db",
			BusyTimeout: 5000,
			PragmaWAL:   true,
		},
		Postgres: PostgresConfig{
			Host:           "127.0.0.1",
			Port:           5433,
			User:           "zuri",
			Password:       "zuri",
			DBName:         "zuri_memory",
			SSLMode:        "disable",
			MaxConnections: 25,
		},
		Qdrant: QdrantConfig{
			Endpoint:   "http://127.0.0.1:6333",
			Collection: "zuri_code_memory",
			TimeoutSec: 10,
		},
	}
}
