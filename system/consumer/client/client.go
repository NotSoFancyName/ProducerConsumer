package client

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/NotSoFancyName/producer-consumer/pkg/proto"
	"github.com/NotSoFancyName/producer-consumer/system/consumer/processor"
)

type Service struct {
	processorService processor.TaskProcessorService
	config           *Config
	logger           *zap.Logger
	limiter          *rate.Limiter

	conn   *grpc.ClientConn
	cancel context.CancelFunc
	ctx    context.Context
}

type Config struct {
	ProducerAddress   string
	RateLimiterConfig RateLimiterConfig
}

type RateLimiterConfig struct {
	Rate  int `yaml:"rate"`
	Burst int `yaml:"burst"`
}

func NewService(processorService processor.TaskProcessorService, config *Config, logger *zap.Logger) (*Service, error) {
	conn, err := grpc.NewClient(config.ProducerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger.Info("Client service rate limiting is set", zap.Int("rate", config.RateLimiterConfig.Rate),
		zap.Int("burst", config.RateLimiterConfig.Burst))

	return &Service{
		processorService: processorService,
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		conn:             conn,
		limiter:          rate.NewLimiter(rate.Limit(config.RateLimiterConfig.Rate), config.RateLimiterConfig.Burst),
	}, nil
}

func (s *Service) Run() {
	unprocessed := s.processorService.GetUnprocessedTasksQueue()
	processed := s.processorService.GetProcessedTasksQueue()
	client := proto.NewTaskProducerClient(s.conn)

	s.logger.Debug("Starting task reader")
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.logger.Debug("Waiting for the rate limiter...")
				err := s.limiter.Wait(context.Background())
				if err != nil {
					s.logger.Error("Rate limiter error", zap.Error(err))
					continue
				}
				s.logger.Debug("Done...")

				s.logger.Debug("Making a task request")
				task, err := client.GetTask(context.Background(), &proto.TaskRequest{})
				if err != nil {
					s.logger.Error("Failed to get task", zap.Error(err))
					continue
				}
				s.logger.Debug("Done...")
				unprocessed <- task.Task.ToModel()
			}
		}
	}()

	s.logger.Debug("Starting task finisher")
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case taskID := <-processed:
				_, err := client.FinishTask(context.Background(), &proto.TaskFinishedRequest{
					Id: uint32(taskID),
				})
				if err != nil {
					s.logger.Error("Error notifying producer finished", zap.Error(err))
				}
			}
		}
	}()
}

func (s *Service) Stop() {
	err := s.conn.Close()
	if err != nil {
		s.logger.Error("Failed to close connection", zap.Error(err))
	}
	s.cancel()
}
