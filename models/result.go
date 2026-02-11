package models

type Result struct {
	WorkerID int
	TaskID   int
	Content  string
	Error    error
}
