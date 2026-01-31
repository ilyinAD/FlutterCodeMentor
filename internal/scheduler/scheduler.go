package scheduler

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/ilyin-ad/flutter-code-mentor/internal/usecase"
	"go.uber.org/zap"
)

type Scheduler struct {
	scheduler gocron.Scheduler
	reviewUC  usecase.ReviewUseCase
	logger    *zap.Logger
}

const kSchedulerDurationJob = 5 * time.Minute

func NewScheduler(reviewUC usecase.ReviewUseCase, logger *zap.Logger) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		scheduler: s,
		reviewUC:  reviewUC,
		logger:    logger,
	}, nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("Starting scheduler")

	_, err := s.scheduler.NewJob(
		gocron.DurationJob(kSchedulerDurationJob),
		gocron.NewTask(func() {
			s.logger.Info("Running scheduled code review task")
			if err := s.reviewUC.ProcessPendingSubmissions(context.Background()); err != nil {
				s.logger.Error("Failed to process pending submissions", zap.Error(err))
			}
		}),
	)

	if err != nil {
		return err
	}

	s.scheduler.Start()
	s.logger.Info("Scheduler started successfully")

	return nil
}

func (s *Scheduler) Stop() error {
	s.logger.Info("Stopping scheduler")
	return s.scheduler.Shutdown()
}
