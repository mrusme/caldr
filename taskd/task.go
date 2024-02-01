package taskd

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type Task struct {
	Description string    `json:"description"`
	Entry       string    `json:"entry"`
	Modified    string    `json:"modified"`
	Scheduled   string    `json:"scheduled"`
	Due         string    `json:"due"`
	End         string    `json:"end"`
	Priority    string    `json:"priority"`
	Project     string    `json:"project"`
	Status      string    `json:"status"`
	UUID        uuid.UUID `json:"uuid"`
	Tags        []string  `json:"tags"`
}

func (t *Task) String() string {
	j, err := json.Marshal(t)
	if err != nil {
		return fmt.Sprintf("{\"error\": \"%s\"}", err)
	}
	return string(j)
}
