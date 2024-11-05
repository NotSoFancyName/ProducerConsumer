package processor

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/NotSoFancyName/producer-consumer/pkg/model"
	"github.com/NotSoFancyName/producer-consumer/pkg/persistence"
)

type TaskProcessorService interface {
	Run()

	GetUnprocessedTasksQueue() chan<- *model.Task
	GetProcessedTasksQueue() <-chan int

	GetStats() *Stats
}

type Stats struct {
	NewTasksDone         int
	NewTasksProcessed    int
	NewTasksPerType      map[int]int
	NewTasksTypeValueSum map[int]int
}

func NewTaskProcessorService(persistenceService persistence.Service, cfg *Config, logger *zap.Logger) (TaskProcessorService, error) {
	return &taskProcessorServiceImpl{
		unprocessedTasksQueue: make(chan *model.Task, cfg.UnprocessedTasksQueueSize),
		processedTasksQueue:   make(chan int, cfg.ProcessedTasksQueueSize),

		newTaskPerType: make(map[int]int),
		newSumPerType:  make(map[int]int),
		sumPerType:     make(map[int]int),

		persistenceService: persistenceService,
		cfg:                cfg,
		logger:             logger,
	}, nil
}

type Config struct {
	ProcessingRoutines        int `yaml:"processing_routines"`
	UnprocessedTasksQueueSize int `yaml:"unprocessed_tasks_queue_size"`
	ProcessedTasksQueueSize   int `yaml:"processed_tasks_queue_size"`
}

type taskProcessorServiceImpl struct {
	persistenceService persistence.Service

	unprocessedTasksQueue chan *model.Task
	processedTasksQueue   chan int

	newTasksDone       int
	newTasksProcessing int
	newTaskPerType     map[int]int
	newSumPerType      map[int]int

	sumPerType map[int]int
	statsMtx   sync.Mutex

	cfg    *Config
	logger *zap.Logger
}

func (s *taskProcessorServiceImpl) GetStats() *Stats {
	s.statsMtx.Lock()
	defer s.statsMtx.Unlock()

	newTasksDone := s.newTasksDone
	newTasksProcessed := s.newTasksProcessing
	newTasksPerType := s.newTaskPerType
	newSumPerType := s.newSumPerType

	s.newTasksDone = 0
	s.newTasksProcessing = 0
	s.newTaskPerType = make(map[int]int)
	s.newSumPerType = make(map[int]int)

	return &Stats{
		NewTasksDone:         newTasksDone,
		NewTasksProcessed:    newTasksProcessed,
		NewTasksTypeValueSum: newSumPerType,
		NewTasksPerType:      newTasksPerType,
	}
}

func (s *taskProcessorServiceImpl) GetProcessedTasksQueue() <-chan int {
	return s.processedTasksQueue
}

func (s *taskProcessorServiceImpl) GetUnprocessedTasksQueue() chan<- *model.Task {
	return s.unprocessedTasksQueue
}

func (s *taskProcessorServiceImpl) Run() {
	for i := 0; i < s.cfg.ProcessingRoutines; i++ {
		go func() {
			for {
				task := <-s.unprocessedTasksQueue
				if err := s.process(task); err != nil {
					s.logger.Error("Error while processing producer", zap.Error(err))
				}
				s.processedTasksQueue <- task.ID
			}
		}()
	}
}

func (s *taskProcessorServiceImpl) process(task *model.Task) error {
	s.statsMtx.Lock()
	s.newTasksProcessing++
	s.statsMtx.Unlock()

	s.logger.Debug("Processing producer", zap.Int("id", task.ID))
	if err := s.persistenceService.UpdateTaskState(task.ID, model.StateProcessing); err != nil {
		s.logger.Error("Error while updating producer state", zap.Error(err))
		return err
	}
	task.State = model.StateProcessing

	time.Sleep(time.Duration(task.Value) * time.Millisecond)

	if err := s.persistenceService.UpdateTaskState(task.ID, model.StateDone); err != nil {
		s.logger.Error("Error while updating producer state", zap.Error(err))
		return err
	}
	task.State = model.StateDone

	s.statsMtx.Lock()
	s.newTasksDone++
	s.newTaskPerType[task.Type]++
	s.newSumPerType[task.Type] += task.Value
	taskTypeSum := s.sumPerType[task.Type] + task.Value
	s.sumPerType[task.Type] = taskTypeSum
	s.statsMtx.Unlock()

	s.logger.Info("Processed producer", zap.Any("producer", task), zap.Int("sum", taskTypeSum))
	return nil
}
