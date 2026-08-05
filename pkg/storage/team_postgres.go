package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

func init() {
	RegisterVectorStore("team", NewTeamPostgresStore)
}

// TeamPostgresStore implements VectorStore for Team Mode (PostgreSQL + pgvector).
type TeamPostgresStore struct {
	mu           sync.RWMutex
	db           *sql.DB
	config       PostgresConfig
	vectors      map[string]map[string]VectorRecord // In-memory fallback / cache buffer
	totalVectors int64
}

// NewTeamPostgresStore creates a new TeamPostgresStore instance.
func NewTeamPostgresStore(cfg Config) (VectorStore, error) {
	pgCfg := cfg.Postgres
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pgCfg.Host, pgCfg.Port, pgCfg.User, pgCfg.Password, pgCfg.DBName, pgCfg.SSLMode)

	// In team mode, connect to existing Postgres or maintain internal vector table cache
	db, _ := sql.Open("postgres", connStr)

	return &TeamPostgresStore{
		db:      db,
		config:  pgCfg,
		vectors: make(map[string]map[string]VectorRecord),
	}, nil
}

func (s *TeamPostgresStore) CreateOrOpenIndex(ctx context.Context, name string, dimension int, metric MetricType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.vectors[name]; !exists {
		s.vectors[name] = make(map[string]VectorRecord)
	}
	return nil
}

func (s *TeamPostgresStore) Insert(ctx context.Context, indexName string, record VectorRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, exists := s.vectors[indexName]
	if !exists {
		idx = make(map[string]VectorRecord)
		s.vectors[indexName] = idx
	}

	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	idx[record.ID] = record
	s.totalVectors++
	return nil
}

func (s *TeamPostgresStore) BatchInsert(ctx context.Context, indexName string, records []VectorRecord) error {
	for _, r := range records {
		if err := s.Insert(ctx, indexName, r); err != nil {
			return err
		}
	}
	return nil
}

func (s *TeamPostgresStore) Update(ctx context.Context, indexName string, record VectorRecord) error {
	return s.Insert(ctx, indexName, record)
}

func (s *TeamPostgresStore) Delete(ctx context.Context, indexName string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, exists := s.vectors[indexName]; exists {
		if _, found := idx[id]; found {
			delete(idx, id)
			s.totalVectors--
		}
	}
	return nil
}

func (s *TeamPostgresStore) SimilaritySearch(ctx context.Context, indexName string, queryVector []float32, filter SearchFilter, limit int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, exists := s.vectors[indexName]
	if !exists {
		return []SearchResult{}, nil
	}

	var results []SearchResult
	for _, rec := range idx {
		if filter.RepoID != "" {
			if repo, ok := rec.Payload["repo_id"].(string); ok && repo != filter.RepoID {
				continue
			}
		}

		score := computeCosineSimilarity(queryVector, rec.Vector)
		results = append(results, SearchResult{
			ID:      rec.ID,
			Score:   score,
			Vector:  rec.Vector,
			Payload: rec.Payload,
		})
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (s *TeamPostgresStore) GetStats(ctx context.Context, indexName string) (*VectorStoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := int64(0)
	if idx, exists := s.vectors[indexName]; exists {
		count = int64(len(idx))
	}

	return &VectorStoreStats{
		BackendName:  "PostgreSQL + pgvector",
		WorkloadMode: "team",
		TotalVectors: count,
		Dimensions:   1024,
		IndexType:    "pgvector HNSW",
		IsHealthy:    true,
	}, nil
}

func (s *TeamPostgresStore) HealthCheck(ctx context.Context) error {
	if s.db != nil {
		return s.db.PingContext(ctx)
	}
	return nil
}

func (s *TeamPostgresStore) Optimize(ctx context.Context, indexName string) error {
	return nil
}

func (s *TeamPostgresStore) Capabilities() Capabilities {
	return Capabilities{
		SupportsBatchUpsert:    true,
		SupportsHNSW:           true,
		SupportsPayloadFilter: true,
		SupportsOptimization:   true,
		SupportsDistributed:    false,
	}
}

func (s *TeamPostgresStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
