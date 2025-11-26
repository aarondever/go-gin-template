package worker

import "context"

type JobHandler func(ctx context.Context, job Job) error

type Job struct {
	Type    string
	Payload map[string]interface{}
	Handler JobHandler
}
