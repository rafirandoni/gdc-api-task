package entity

import (
	"strings"
)

type TaskStatus string

const (
	TaskBacklog    TaskStatus = "BACKLOG"
	TaskOnProgress TaskStatus = "ON_PROGRESS"
	TaskOnReview   TaskStatus = "ON_REVIEW"
	TaskFeedback   TaskStatus = "FEEDBACK"
	TaskFinished   TaskStatus = "FINISHED"
)

func ParseTaskStatus(raw string) (TaskStatus, bool) {
	status := TaskStatus(strings.ToUpper(strings.TrimSpace(raw)))
	switch status {
	case TaskBacklog, TaskOnProgress, TaskOnReview, TaskFeedback, TaskFinished:
		return status, true
	default:
		return "", false
	}
}
