package producer

import (
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/NotSoFancyName/producer-consumer/pkg/model"
	"github.com/NotSoFancyName/producer-consumer/pkg/persistence"
)

const (
	maxTaskTypeNumber  = 10
	maxTaskValueNumber = 100
)

var (
	ErrToManyPendingTasks = errors.New("to many pending tasks")
)

type Config struct {
	// tasks produced per second
	TaskProductionRate int `yaml:"task_production_rate"`
	MaxPendingTasks    int `yaml:"max_pending_tasks"`
}

type Stats struct {
	NewlyProducedTasks int
}

type Service interface {
	Run()
	GetTaskQueue() <-chan *model.Task
	FinishTask(id int) error
	GetStats() (*Stats, error)
}

func NewService(persistenceService persistence.Service, config *Config, logger *zap.Logger) (Service, error) {
	if config.TaskProductionRate <= 0 {
		config.TaskProductionRate = 1
	}

	productionPeriod := time.Second / time.Duration(config.TaskProductionRate)
	if productionPeriod <= 0 {
		productionPeriod = time.Nanosecond
	}

	logger.Info("Producer service initialized", zap.Duration("productionPeriod", productionPeriod))

	return &producerServiceImpl{
		cfg: config,

		persistenceService: persistenceService,
		producedTasks:      make(chan *model.Task, config.MaxPendingTasks),
		unprocessedTasks:   map[int]struct{}{},
		producerTicker:     time.NewTicker(productionPeriod),
		logger:             logger,
	}, nil
}

type producerServiceImpl struct {
	cfg *Config

	producedTasks chan *model.Task

	newProducedTasks    int
	newProducedTasksMtx sync.Mutex

	unprocessedTasks    map[int]struct{}
	unprocessedTasksMtx sync.Mutex

	producerTicker *time.Ticker

	persistenceService persistence.Service

	logger *zap.Logger
}

func (p *producerServiceImpl) Run() {
	go func() {
		for {
			select {
			case <-p.producerTicker.C:
				task, err := p.produceTask()
				if err != nil {
					if !errors.Is(err, ErrToManyPendingTasks) {
						p.logger.Error("Failed to produce producer", zap.Error(err))
					}
					continue
				}
				p.producedTasks <- task
			}
		}
	}()

}

func (p *producerServiceImpl) GetTaskQueue() <-chan *model.Task {
	return p.producedTasks
}

func (p *producerServiceImpl) FinishTask(id int) error {
	p.unprocessedTasksMtx.Lock()
	defer p.unprocessedTasksMtx.Unlock()

	delete(p.unprocessedTasks, id)
	return nil
}

func (p *producerServiceImpl) GetStats() (*Stats, error) {
	p.newProducedTasksMtx.Lock()
	defer p.newProducedTasksMtx.Unlock()

	newlyProducedTasks := p.newProducedTasks
	p.newProducedTasks = 0

	return &Stats{
		NewlyProducedTasks: newlyProducedTasks,
	}, nil
}

func (p *producerServiceImpl) produceTask() (*model.Task, error) {
	p.unprocessedTasksMtx.Lock()
	defer p.unprocessedTasksMtx.Unlock()

	if len(p.unprocessedTasks) >= p.cfg.MaxPendingTasks {
		return nil, ErrToManyPendingTasks
	}

	taskType := rand.IntN(maxTaskTypeNumber)
	taskValue := rand.IntN(maxTaskValueNumber)

	createdTask, err := p.persistenceService.CreateTask(&model.Task{
		Type:  taskType,
		Value: taskValue,
		State: model.StateReceived,
	})
	if err != nil {
		return nil, err
	}
	p.unprocessedTasks[createdTask.ID] = struct{}{}

	p.newProducedTasksMtx.Lock()
	p.newProducedTasks++
	p.newProducedTasksMtx.Unlock()

	return createdTask, nil
}
