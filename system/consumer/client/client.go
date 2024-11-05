package client

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/NotSoFancyName/producer-consumer/pkg/proto"
	"github.com/NotSoFancyName/producer-consumer/system/consumer/processor"
)

type Service struct {
	processorService processor.TaskProcessorService
	config           *Config
	logger           *zap.Logger

	conn   *grpc.ClientConn
	cancel context.CancelFunc
	ctx    context.Context
}

func NewService(processorService processor.TaskProcessorService, config *Config, logger *zap.Logger) (*Service, error) {
	conn, err := grpc.NewClient(config.ProducerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		processorService: processorService,
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		conn:             conn,
	}, nil
}

type Config struct {
	ProducerAddress string
}

func (s *Service) Run() {

	//limiter := rate.NewLimiter(rate.Limit(1000), 5000)

	unprocessed := s.processorService.GetUnprocessedTasksQueue()
	processed := s.processorService.GetProcessedTasksQueue()
	client := proto.NewTaskProducerClient(s.conn)

	go func() {
		stream, err := client.GetTask(context.Background(), &proto.TaskRequest{})
		if err != nil {
			s.logger.Error("failed to get tasks", zap.Error(err))
		}

		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				//err := limiter.Wait(context.Background())
				//if err != nil {
				//	s.logger.Error("rate limiter error", zap.Error(err))
				//	continue
				//}

				task, err := stream.Recv()
				if err != nil {
					s.logger.Error("failed to get tasks", zap.Error(err))
					continue
				}
				unprocessed <- task.Task.ToModel()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case taskID := <-processed:
				_, err := client.NotifyTaskFinished(context.Background(), &proto.TaskFinishedRequest{
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
