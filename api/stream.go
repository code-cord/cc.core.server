package api

import "context"

// Stream represents stream API.
type Stream interface {
	Start(ctx context.Context) error
	//Info(ctx context.Context)
	//Stop(ctx context.Context) error
}
