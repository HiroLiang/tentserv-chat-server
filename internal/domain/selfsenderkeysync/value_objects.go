package selfsenderkeysync

type Status string

const (
	StatusPendingProvider Status = "pending_provider"
	StatusSyncing         Status = "syncing"
	StatusUploaded        Status = "uploaded"
	StatusCompleted       Status = "completed"
	StatusFailed          Status = "failed"
)
