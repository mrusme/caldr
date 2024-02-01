package main

import "github.com/mrusme/caldr/taskd"

func runTaskd() error {
	td, err := taskd.New(taskdPort, taskdCertFile, taskdKeyFile, taskdProcessor)
	if err != nil {
		return err
	}

	err = td.Launch()
	if err != nil {
		return err
	}

	return nil
}

func taskdProcessor(newSyncID string, msg taskd.Message) (taskd.Message, error) {
	return taskd.Message{}, nil
}
