package server

import (
	"context"
	"go.uber.org/zap"

	"github.com/NotSoFancyName/producer-consumer/pkg/proto"
	"github.com/NotSoFancyName/producer-consumer/system/producer/producer"
)

type GRPCTaskService struct {
	proto.UnimplementedTaskProducerServer

	producerService producer.Service
	logger          *zap.Logger
}

func NewGRPCTaskService(producerService producer.Service, logger *zap.Logger) (*GRPCTaskService, error) {
	return &GRPCTaskService{
		producerService: producerService,
		logger:          logger,
	}, nil
}

func (s *GRPCTaskService) GetTask(ctx context.Context, _ *proto.TaskRequest) (*proto.TaskResponse, error) {
	s.logger.Debug("Received task request")
	task := <-s.producerService.GetTaskQueue()
	s.logger.Debug("Received task response from task producer")

	return &proto.TaskResponse{
		Task: proto.FromModel(task),
	}, nil
}

func (s *GRPCTaskService) FinishTask(ctx context.Context, request *proto.TaskFinishedRequest) (*proto.TaskFinishedResponse, error) {
	err := s.producerService.FinishTask(int(request.GetId()))
	if err != nil {
		s.logger.Error("Error sending producer to consumer", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("Acknowledged  message processed", zap.Int("id", int(request.GetId())))
	return &proto.TaskFinishedResponse{}, nil
}
