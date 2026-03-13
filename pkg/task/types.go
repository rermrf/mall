package task

import "context"

type Task interface {
	Start(ctx context.Context)
}
