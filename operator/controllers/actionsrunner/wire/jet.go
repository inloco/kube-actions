package wire

type JobEventType string

const (
	JobAssigned  JobEventType = "JobAssigned"
	JobCompleted JobEventType = "JobCompleted"
	JobStarted   JobEventType = "JobStarted"
)

func (jet JobEventType) StringReference() *string {
	s := string(jet)
	return &s
}
