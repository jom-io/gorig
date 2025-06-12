package cronx

//type localTask[t any] struct {
//	ID         string          `json:"id"`         // task ID
//	TaskID     string          `json:"taskID"`     // task unique identifier
//	Name       string          `json:"name"`       // task name
//	LastRunAt  time.Time       `json:"lastRunAt"`  // last run time
//	RunAt      time.Time       `json:"runAt"`      // next run time
//	Params     t               `json:"params"`     // task parameters
//	Status     LocalTaskStatus `json:"status"`     // task status
//	Type       LocalTaskType   `json:"type"`       // task type (cron or once)
//	Method     string          `json:"method"`     // method name to execute
//	RetryCount int             `json:"retryCount"` // number of retries
//	MaxRetries int             `json:"maxRetries"` // maximum number of retries
//	RetryDelay []time.Duration `json:"retryDelay"` // delay between retries
//}
//
//type LocalTaskType string
//
//const (
//	TaskTypeCron LocalTaskType = "cron" // Cron task
//	TaskTypeOnce LocalTaskType = "once" // One-time task
//)
//
//type LocalTaskStatus string
//
//const (
//	TaskStatusPending   LocalTaskStatus = "pending"   // Task is pending
//	TaskStatusRunning   LocalTaskStatus = "running"   // Task is currently running
//	TaskStatusDone      LocalTaskStatus = "done"      // Task has completed successfully
//	TaskStatusFailed    LocalTaskStatus = "failed"    // Task has failed
//	TaskStatusCancelled LocalTaskStatus = "cancelled" // Task has been cancelled
//	TaskStatusRetrying  LocalTaskStatus = "retrying"  // Task is retrying after failure
//)
//
//type TaskStore interface {
//	Add(task *localTask[any]) error
//	Get(id string) (*localTask[any], error)
//	ListPending() ([]*localTask[any], error)
//	UpdateStatus(id string, status LocalTaskStatus) error
//	IncrementRetry(id string) error
//	Delete(id string) error
//}
//
//type Scheduler interface {
//	Schedule(task *localTask[any], exec func(func(t *localTask[any]) error)) error // Schedule a task
//	Cancel(id string) error                                                        // Cancel a scheduled task
//}
//
//type TaskExecutor interface {
//	Execute(task *localTask[any]) error // Execute a task
//}
