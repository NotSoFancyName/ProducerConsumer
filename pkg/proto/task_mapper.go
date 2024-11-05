package proto

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/NotSoFancyName/producer-consumer/pkg/model"
)

func (t *Task) ToModel() *model.Task {
	return &model.Task{
		ID:             int(t.GetId()),
		Type:           int(t.GetType()),
		Value:          int(t.GetValue()),
		State:          model.StateType(StateType_name[int32(t.GetState())]),
		CreationTime:   t.GetCreationTime().AsTime(),
		LastUpdateTime: t.GetLastUpdateTime().AsTime(),
	}
}

func FromModel(t *model.Task) *Task {
	return &Task{
		Id:             uint32(t.ID),
		Type:           uint32(t.Type),
		Value:          uint32(t.Value),
		State:          StateType(StateType_value[string(t.State)]),
		CreationTime:   timestamppb.New(t.CreationTime),
		LastUpdateTime: timestamppb.New(t.LastUpdateTime),
	}
}
