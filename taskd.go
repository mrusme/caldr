package main

import (
	"fmt"

	"github.com/emersion/go-ical"
	"github.com/mrusme/caldr/store"
	"github.com/mrusme/caldr/taskd"
)

func runTaskd(db *store.Store) error {
	td, err := taskd.New(taskdPort, taskdCertFile, taskdKeyFile, taskdProcessor(db))
	if err != nil {
		return err
	}

	err = td.Launch()
	if err != nil {
		return err
	}

	return nil
}

func taskdProcessor(db *store.Store) taskd.Processor {
	return func(newSyncID string, msg taskd.Message) (taskd.Message, error) {
		todos, err := db.ListTodos()
		if err != nil {
			return taskd.Message{}, err
		}

		fmt.Printf("\n\nTODOS:\n\n%#v\n", todos)

		var tasks []taskd.Task
		for _, todo := range todos {
			var task taskd.Task = taskd.Task{
				Due: store.GetPropValueSafe(&todo.Props, ical.PropDue),
			}

			tasks = append(tasks, task)
		}

		fmt.Printf("\n\nTASKS:\n\n%#v\n", tasks)

		return taskd.Message{
			Tasks: tasks,
		}, nil
	}
}
