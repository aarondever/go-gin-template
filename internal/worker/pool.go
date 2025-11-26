package worker

import (
	"context"
	"sync"

	"github.com/aarondever/go-gin-template/pkg/logger"
)

type Pool struct {
	workers  int
	jobQueue chan Job
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewPool(workers, queueSize int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		workers:  workers,
		jobQueue: make(chan Job, queueSize),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()

	logger.Info("Worker started", "worker_id", id)

	for {
		select {
		case <-p.ctx.Done():
			logger.Info("Worker stopped", "worker_id", id)
			return
		case job, ok := <-p.jobQueue:
			if !ok {
				logger.Info("Worker channel closed", "worker_id", id)
				return
			}

			logger.Debug("Worker processing job", "worker_id", id, "job_type", job.Type)

			if err := job.Handler(p.ctx, job); err != nil {
				logger.Error("Job failed", "worker_id", id, "job_type", job.Type, "error", err)
			} else {
				logger.Debug("Job completed", "worker_id", id, "job_type", job.Type)
			}
		}
	}
}

func (p *Pool) Submit(job Job) {
	select {
	case p.jobQueue <- job:
		logger.Debug("Job submitted", "job_type", job.Type)
	case <-p.ctx.Done():
		logger.Warn("Cannot submit job, pool is stopped", "job_type", job.Type)
	default:
		logger.Warn("Job queue full, dropping job", "job_type", job.Type)
	}
}

func (p *Pool) Stop() {
	logger.Info("Stopping worker pool...")
	p.cancel()
	close(p.jobQueue)
	p.wg.Wait()
	logger.Info("Worker pool stopped")
}
