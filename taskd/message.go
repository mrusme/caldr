package taskd

type Message struct {
	Client   string
	Protocol string
	Type     string
	Org      string
	User     string
	Key      string
	SyncID   string
	Tasks    []Task
}
