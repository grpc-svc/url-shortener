package instrumented

import (
	"context"
	"time"
	"url-shortener/internal/lib/metrics"
	"url-shortener/internal/storage"
)

type Storage struct {
	next storage.Storage
}

func New(next storage.Storage) *Storage {
	return &Storage{next: next}
}

func (s *Storage) SaveURL(ctx context.Context, alias, originalURL, ownerEmail string) error {
	const op = "SaveURL"
	start := time.Now()
	err := s.next.SaveURL(ctx, alias, originalURL, ownerEmail)
	s.recordMetrics(op, err, start)
	return err
}
func (s *Storage) GetURL(ctx context.Context, alias string) (string, error) {
	const op = "GetURL"
	start := time.Now()
	url, err := s.next.GetURL(ctx, alias)
	s.recordMetrics(op, err, start)
	return url, err
}
func (s *Storage) GetURLOwner(ctx context.Context, alias string) (string, error) {
	const op = "GetURLOwner"
	start := time.Now()
	owner, err := s.next.GetURLOwner(ctx, alias)
	s.recordMetrics(op, err, start)
	return owner, err
}
func (s *Storage) DeleteURL(ctx context.Context, alias string) error {
	const op = "DeleteURL"
	start := time.Now()
	err := s.next.DeleteURL(ctx, alias)
	s.recordMetrics(op, err, start)
	return err
}
func (s *Storage) Close() error {
	return s.next.Close()
}
func (s *Storage) recordMetrics(operation string, err error, start time.Time) {
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	metrics.StorageOperationsTotal.WithLabelValues(operation, status).Inc()
	metrics.StorageOperationDuration.WithLabelValues(operation).Observe(duration)
}
