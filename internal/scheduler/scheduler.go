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
	cancel    context.CancelFunc
}

const kSchedulerDurationJob = 1 * time.Minute
const kScheduleStartTimeJob = 10 * time.Second

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

func (s *Scheduler) Start(_ context.Context) error {
	s.logger.Info("Starting scheduler")

	appCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	_, err := s.scheduler.NewJob(
		gocron.DurationJob(kSchedulerDurationJob),
		gocron.NewTask(func() {
			s.logger.Info("Running scheduled code review task")
			jobCtx, jobCancel := context.WithTimeout(appCtx, 45*time.Minute)
			defer jobCancel()
			if err := s.reviewUC.ProcessPendingSubmissions(jobCtx); err != nil {
				s.logger.Error("Failed to process pending submissions", zap.Error(err))
			}
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithStartAt(gocron.WithStartDateTime(time.Now().Add(kScheduleStartTimeJob))),
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
	if s.cancel != nil {
		s.cancel()
	}
	return s.scheduler.Shutdown()
}
