package scheduler

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

type Job interface {
	Run()
}

type Scheduler struct {
	*gocron.Scheduler
}

func NewScheduler() *Scheduler {
	return &Scheduler{gocron.NewScheduler(time.Local)}
}

func (s *Scheduler) Add(schedule string, job Job) error {
	_, err := s.Cron(schedule).Do(job.Run)
	return err
}

func (s *Scheduler) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.Stop()
		logrus.WithContext(ctx).Debug("scheduler stopped")
	}()

	logrus.WithContext(ctx).Debug("scheduler started")
	s.StartBlocking()
}
