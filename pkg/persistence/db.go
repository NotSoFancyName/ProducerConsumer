package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/NotSoFancyName/producer-consumer/pkg/model"
	"github.com/NotSoFancyName/producer-consumer/pkg/persistence/db"
)

const (
	postgresDriverName = "postgres"
)

type Config struct {
	Host                  string        `yaml:"host"`
	Port                  int           `yaml:"port"`
	Username              string        `yaml:"username"`
	DBName                string        `yaml:"db_name"`
	Password              string        `yaml:"password"`
	ConnectionMaxLifetime time.Duration `yaml:"connection_max_lifetime"`
	MaxIdleConnections    int           `yaml:"max_idle_connections"`
	MaxOpenConnections    int           `yaml:"max_open_connections"`
}

type Service interface {
	GetTask(id int) (*model.Task, error)
	CreateTask(task *model.Task) (*model.Task, error)
	UpdateTaskState(id int, state model.StateType) error
}

func NewService(config *Config) (Service, error) {
	dsn := fmt.Sprintf("host=%s port=%v user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.Username, config.Password, config.DBName)
	database, err := sql.Open(postgresDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}
	database.SetConnMaxLifetime(config.ConnectionMaxLifetime)
	database.SetMaxIdleConns(config.MaxIdleConnections)
	database.SetMaxOpenConns(config.MaxOpenConnections)

	return &persistenceServiceImpl{
		db:      database,
		queries: db.New(database),
	}, nil
}

type persistenceServiceImpl struct {
	db      *sql.DB
	queries *db.Queries
}

func (s *persistenceServiceImpl) GetTask(id int) (*model.Task, error) {
	task, err := s.queries.GetTask(context.Background(), int64(id))
	if err != nil {
		return nil, fmt.Errorf("error getting producer: %v", err)
	}

	return model.TaskModel(task), nil
}

func (s *persistenceServiceImpl) CreateTask(task *model.Task) (*model.Task, error) {
	createdTask, err := s.queries.CreateTask(context.Background(), db.CreateTaskParams{
		Type:  sql.NullInt32{Valid: true, Int32: int32(task.Type)},
		Value: sql.NullInt32{Valid: true, Int32: int32(task.Value)},
		State: task.State.DBModel(),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating producer: %v", err)
	}

	return model.TaskModel(createdTask), nil
}

func (s *persistenceServiceImpl) UpdateTaskState(id int, state model.StateType) error {
	err := s.queries.UpdateTask(context.Background(), db.UpdateTaskParams{
		ID:    int64(id),
		State: state.DBModel(),
	})
	if err != nil {
		return fmt.Errorf("error creating producer: %v", err)
	}
	return nil
}
