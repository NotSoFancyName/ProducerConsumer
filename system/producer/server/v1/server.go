package v1

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"

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

func (s *GRPCTaskService) GetTask(_ *proto.TaskRequest, stream proto.TaskProducer_GetTaskServer) error {
	producedTasksQueue := s.producerService.GetTaskQueue()
	for {
		acquiredTask := <-producedTasksQueue
		err := stream.Send(&proto.TaskResponse{
			Task: proto.FromModel(acquiredTask),
		})
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Unavailable {
				s.logger.Info("Client connection closed", zap.Error(err))
				return nil
			}
			s.logger.Error("Error sending producer to consumer", zap.Error(err))
			continue
		}
	}
}

func (s *GRPCTaskService) NotifyTaskFinished(ctx context.Context, request *proto.TaskFinishedRequest) (*proto.TaskFinishedResponse, error) {
	err := s.producerService.FinishTask(int(request.GetId()))
	if err != nil {
		s.logger.Error("Error sending producer to consumer", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("Acknowledged  message processed", zap.Int("id", int(request.GetId())))
	return &proto.TaskFinishedResponse{}, nil
}
