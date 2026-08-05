package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricType defines vector distance calculation metrics.
type MetricType string

const (
	MetricCosine MetricType = "cosine"
	MetricDot    MetricType = "dot"
	MetricL2     MetricType = "l2"
)

// VectorRecord represents a vector embedding payload.
type VectorRecord struct {
	ID        string                 `json:"id"`
	Vector    []float32              `json:"vector"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
}

// SearchFilter specifies metadata filtering parameters for vector queries.
type SearchFilter struct {
	RepoID        string   `json:"repo_id,omitempty"`
	Boundary      string   `json:"boundary,omitempty"`
	Concern       string   `json:"concern,omitempty"`
	DecisionType  string   `json:"decision_type,omitempty"`
	MinConfidence float64  `json:"min_confidence,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

// SearchResult represents a match returned by vector similarity search.
type SearchResult struct {
	ID        string                 `json:"id"`
	Score     float32                `json:"score"`
	Vector    []float32              `json:"vector,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
}

// VectorStoreStats contains health metrics and index metadata.
type VectorStoreStats struct {
	BackendName string    `json:"backend_name"`
	WorkloadMode string   `json:"workload_mode"`
	TotalVectors int64     `json:"total_vectors"`
	Dimensions   int       `json:"dimensions"`
	IndexType    string    `json:"index_type"`
	IsHealthy    bool      `json:"is_healthy"`
	LastOptimized *time.Time `json:"last_optimized,omitempty"`
}

// Capabilities defines feature support flags advertised by a VectorStore backend.
type Capabilities struct {
	SupportsBatchUpsert    bool `json:"supports_batch_upsert"`
	SupportsHNSW           bool `json:"supports_hnsw"`
	SupportsPayloadFilter bool `json:"supports_payload_filter"`
	SupportsOptimization   bool `json:"supports_optimization"`
	SupportsDistributed    bool `json:"supports_distributed"`
}

// VectorStore defines the storage-agnostic contract for all vector database implementations.
type VectorStore interface {
	// CreateOrOpenIndex creates or opens a vector index collection.
	CreateOrOpenIndex(ctx context.Context, name string, dimension int, metric MetricType) error

	// Insert stores a single vector record.
	Insert(ctx context.Context, indexName string, record VectorRecord) error

	// BatchInsert stores multiple vector records in a single batch operation.
	BatchInsert(ctx context.Context, indexName string, records []VectorRecord) error

	// Update updates an existing vector record payload or vector embedding.
	Update(ctx context.Context, indexName string, record VectorRecord) error

	// Delete removes a vector record by ID.
	Delete(ctx context.Context, indexName string, id string) error

	// SimilaritySearch retrieves top-K nearest neighbors matching query vector and filter.
	SimilaritySearch(ctx context.Context, indexName string, queryVector []float32, filter SearchFilter, limit int) ([]SearchResult, error)

	// GetStats returns current store operational stats.
	GetStats(ctx context.Context, indexName string) (*VectorStoreStats, error)

	// HealthCheck verifies backend network or database connection state.
	HealthCheck(ctx context.Context) error

	// Optimize triggers backend index maintenance or compaction when supported.
	Optimize(ctx context.Context, indexName string) error

	// Capabilities advertises supported features of the underlying vector store.
	Capabilities() Capabilities

	// Close gracefully closes database connections or network sockets.
	Close() error
}

// Factory constructor registration pattern
type FactoryFunc func(config Config) (VectorStore, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]FactoryFunc)
)

// RegisterVectorStore registers a vector store backend constructor.
func RegisterVectorStore(name string, factory FactoryFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if name == "" {
		panic("storage: cannot register VectorStore with empty name")
	}
	if factory == nil {
		panic("storage: cannot register nil VectorStore factory")
	}
	registry[name] = factory
}

// NewVectorStore constructs a VectorStore instance based on workload mode configuration.
func NewVectorStore(cfg Config) (VectorStore, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	mode := cfg.Mode
	if mode == "" {
		mode = "local"
	}

	factory, exists := registry[mode]
	if !exists {
		return nil, fmt.Errorf("storage: unregistered workload mode '%s'. Available modes: local, team, enterprise", mode)
	}

	store, err := factory(cfg)
	if err != nil {
		return nil, fmt.Errorf("storage: initializing vector store for mode '%s': %w", mode, err)
	}

	return store, nil
}
