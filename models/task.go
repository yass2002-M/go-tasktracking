package models

import "time"

type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Deadline    time.Time `json:"deadline"`
	Importance  int       `json:"importance"`
	WorkMinutes int       `json:"work_minutes"`
	WorkSeconds int       `json:"work_seconds,omitempty"`
	Status      string    `json:"status"` // "general", "today", "late"
	Note        string    `json:"note"`   // note facultative
	Subtasks    []Subtask `json:"subtasks,omitempty"`
}

type Subtask struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}
