package cleaner

import "context"

type Result struct {
	Bytes int64
	Items int
	Path  string
}

type Options struct {
	DryRun bool
	Yes    bool
}

type Cleaner interface {
	ID() string
	Name() string
	Category() string
	Available(ctx context.Context) bool
	Scan(ctx context.Context) (Result, error)
	Clean(ctx context.Context, opts Options) (Result, error)
}
