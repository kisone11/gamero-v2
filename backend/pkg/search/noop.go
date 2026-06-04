package search

import (
	"context"
	"errors"
)

// NoopSearcher 空实现，ES 未启用时使用，搜索由 PostgreSQL FTS 负责
type NoopSearcher struct{}

func NewNoopSearcher() *NoopSearcher { return &NoopSearcher{} }

var errESDisabled = errors.New("search: ES not enabled, using PostgreSQL FTS")

func (n *NoopSearcher) IndexProject(_ context.Context, _ ProjectDoc) error { return nil }
func (n *NoopSearcher) IndexUser(_ context.Context, _ UserDoc) error       { return nil }
func (n *NoopSearcher) IndexLog(_ context.Context, _ LogDoc) error         { return nil }
func (n *NoopSearcher) IndexPost(_ context.Context, _ PostDoc) error       { return nil }
func (n *NoopSearcher) DeleteProject(_ context.Context, _ uint64) error    { return nil }
func (n *NoopSearcher) DeleteUser(_ context.Context, _ uint64) error       { return nil }
func (n *NoopSearcher) DeleteLog(_ context.Context, _ uint64) error        { return nil }
func (n *NoopSearcher) DeletePost(_ context.Context, _ uint64) error       { return nil }
func (n *NoopSearcher) Ping(_ context.Context) error                       { return nil }

func (n *NoopSearcher) SearchProjects(_ context.Context, _ string, _, _ int) (Result, error) {
	return Result{}, errESDisabled
}
func (n *NoopSearcher) SearchUsers(_ context.Context, _ string, _, _ int) (Result, error) {
	return Result{}, errESDisabled
}
func (n *NoopSearcher) SearchLogs(_ context.Context, _ string, _, _ int) (Result, error) {
	return Result{}, errESDisabled
}
func (n *NoopSearcher) SearchPosts(_ context.Context, _ string, _, _ int) (Result, error) {
	return Result{}, errESDisabled
}
