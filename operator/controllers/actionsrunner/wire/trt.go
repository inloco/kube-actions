package wire

type TimelineRecordType string

const (
	JobTimelineRecordType  TimelineRecordType = "Job"
	TaskTimelineRecordType TimelineRecordType = "Task"
)

func (trt TimelineRecordType) StringReference() *string {
	s := string(trt)
	return &s
}
