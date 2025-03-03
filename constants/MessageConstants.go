package constants

// Task-related constants and messages.
const (
	ERROR_LOADING_TASKS         = "Error loading tasks"
	ERROR_CREATING_TASK_MANAGER = "Error creating task manager"
)

const (
	CREATE_TASK   = "CreateTask"
	COMPLETE_TASK = "CompleteTask"
	DELETE_TASK   = "DeleteTask"
	LIST_TASK     = "ListTask"
)

// Error Messages for tasks and general errors.
const (
	ErrorInvalidMethod          = "Invalid request method"
	ErrorInvalidInput           = "Invalid input"
	ErrorUnauthorized           = "Unauthorized access"
	ErrorDatabaseInitialization = "Error initializing database connection"
	ErrorGeneratingToken        = "Error generating token"
	ErrorInvalidCredentials     = "Invalid credentials"
	ErrorTaskNotFound           = "Task not found"
)

// Success Messages for tasks.
const (
	SuccessTaskCompleted = "Task marked as complete"
	SuccessTaskDeleted   = "Task successfully deleted"
	SuccessTaskAdded     = "Task successfully added"
)

// Image Event constants.
const (
	IMAGE_UPLOAD       = "ImageUpload"
	IMAGE_UPDATE       = "ImageUpdate"
	IMAGE_DELETE       = "ImageDelete"
	IMAGE_STATS_UPDATE = "ImageStatsUpdate"
)

// Error Messages for image events.
const (
	ErrorImageUpload      = "Error uploading image"
	ErrorImageUpdate      = "Error updating image"
	ErrorImageStatsUpdate = "Error updating image statistics"
)
