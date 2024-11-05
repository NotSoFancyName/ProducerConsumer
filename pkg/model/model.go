package model

import (
	"database/sql"
	"time"

	"github.com/NotSoFancyName/producer-consumer/pkg/persistence/db"
)

type StateType string

const (
	StateUnknown    StateType = "UNKNOWN_STATE"
	StateReceived   StateType = "RECEIVED"
	StateProcessing StateType = "PROCESSING"
	StateDone       StateType = "DONE"
)

type Task struct {
	ID             int
	Type           int
	Value          int
	State          StateType
	CreationTime   time.Time
	LastUpdateTime time.Time
}

func (s StateType) DBModel() db.NullTaskState {
	switch s {
	case StateReceived:
		return db.NullTaskState{
			TaskState: db.TaskStateReceived,
			Valid:     true,
		}
	case StateProcessing:
		return db.NullTaskState{
			TaskState: db.TaskStateProcessing,
			Valid:     true,
		}
	case StateDone:
		return db.NullTaskState{
			TaskState: db.TaskStateDone,
			Valid:     true,
		}
	default:
		return db.NullTaskState{}
	}
}

func StateModel(state db.NullTaskState) StateType {
	switch state.TaskState {
	case db.TaskStateReceived:
		return StateReceived
	case db.TaskStateProcessing:
		return StateProcessing
	case db.TaskStateDone:
		return StateDone
	default:
		return StateUnknown
	}
}

func (t *Task) DBModel() db.Task {
	return db.Task{
		ID: int64(t.ID),
		Type: sql.NullInt32{
			Int32: int32(t.Type),
			Valid: true,
		},
		Value: sql.NullInt32{
			Int32: int32(t.Value),
			Valid: true,
		},
		State: t.State.DBModel(),
		CreationTime: sql.NullTime{
			Time:  t.CreationTime,
			Valid: true,
		},
		LastUpdateTime: sql.NullTime{
			Time:  t.LastUpdateTime,
			Valid: true,
		},
	}
}

func TaskModel(task db.Task) *Task {
	var creationTime time.Time
	if task.CreationTime.Valid {
		creationTime = task.CreationTime.Time
	}

	var lastUpdateTime time.Time
	if task.LastUpdateTime.Valid {
		lastUpdateTime = task.LastUpdateTime.Time
	}

	return &Task{
		ID:             int(task.ID),
		Type:           int(task.Type.Int32),
		Value:          int(task.Value.Int32),
		State:          StateModel(task.State),
		CreationTime:   creationTime,
		LastUpdateTime: lastUpdateTime,
	}
}
