package lesson

import (
	"time"
)

// VersionInfo represents information about a specific version of a lesson.
// It includes the version number, timestamp, and a description of changes.
type VersionInfo struct {
	// Version is the version number
	Version int `json:"version" bson:"version"`

	// Timestamp is when this version was created
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`

	// ChangeSummary is a brief description of changes in this version
	ChangeSummary string `json:"change_summary" bson:"change_summary"`
}

// ContainerConfig represents the configuration for a single container in a multi-container environment.
// It includes the image to use, resource limits, and other container-specific settings.
type ContainerConfig struct {
	// Name is a unique identifier for the container within the step
	Name string `json:"name" bson:"name"`

	// Image is the Docker image to use for this container
	Image string `json:"image" bson:"image"`

	// Role defines the purpose of this container in the environment (e.g., "primary", "database", "cache")
	Role string `json:"role" bson:"role"`

	// Hostname is the hostname to assign to the container
	Hostname string `json:"hostname" bson:"hostname"`

	// Ports is a list of ports to expose from the container
	Ports []string `json:"ports,omitempty" bson:"ports,omitempty"`

	// Envs is a list of environment variables to set in the container
	Envs []string `json:"envs,omitempty" bson:"envs,omitempty"`

	// Networks is a list of additional networks to connect the container to
	Networks []string `json:"networks,omitempty" bson:"networks,omitempty"`

	// Resource limits for this container

	// MaxProcesses is the maximum number of processes that can be created in the container
	MaxProcesses int64 `json:"max_processes,omitempty" bson:"max_processes,omitempty"`

	// MaxMemoryMB is the maximum amount of memory that the container can use, in megabytes
	MaxMemoryMB int64 `json:"max_memory_mb,omitempty" bson:"max_memory_mb,omitempty"`

	// StorageSize is the maximum amount of storage that the container can use
	StorageSize string `json:"storage_size,omitempty" bson:"storage_size,omitempty"`
}

// LessonStep represents a single step in a lesson.
// Each step contains content to be displayed to the user, commands to be executed,
// expected output for validation, and other metadata.
type LessonStep struct {
	// ID is a unique identifier for the step
	ID string `json:"id" bson:"id"`

	// Content is the markdown content to be displayed to the user
	Content string `json:"content" bson:"content"`

	// Commands is a list of shell commands that can be executed in the lesson environment
	Commands []string `json:"commands" bson:"commands"`

	// Expected is the expected output of the commands, used for validation
	Expected string `json:"expected" bson:"expected"`

	// Image is the Docker image to use for this step (if different from the lesson default)
	// This field is maintained for backward compatibility with single-container environments
	Image string `json:"image" bson:"image"`

	// Timeout is the maximum duration allowed for this step to complete
	Timeout time.Duration `json:"timeout" bson:"timeout"`

	// Question is an optional question to be displayed to the user
	Question string `json:"question" bson:"question"`

	// Resource limits for this step (if different from the lesson defaults)
	// These apply to the primary container when using a single-container environment

	// MaxProcesses is the maximum number of processes that can be created in the container
	MaxProcesses int64 `json:"max_processes,omitempty" bson:"max_processes,omitempty"`

	// MaxMemoryMB is the maximum amount of memory that the container can use, in megabytes
	MaxMemoryMB int64 `json:"max_memory_mb,omitempty" bson:"max_memory_mb,omitempty"`

	// StorageSize is the maximum amount of storage that the container can use
	StorageSize string `json:"storage_size,omitempty" bson:"storage_size,omitempty"`

	// Containers is a list of container configurations for multi-container environments
	// If this field is empty, a single container will be created using the Image field
	Containers []ContainerConfig `json:"containers,omitempty" bson:"containers,omitempty"`
}

// Lesson represents a complete lesson with multiple steps.
// A lesson is a structured learning experience that guides users through
// a series of steps, each with its own content, commands, and validation.
type Lesson struct {
	// ID is a unique identifier for the lesson
	ID string `json:"id" bson:"id"`

	// Title is the title of the lesson
	Title string `json:"title" bson:"title"`

	// Description is a brief description of the lesson
	Description string `json:"description" bson:"description"`

	// Category is the primary category of the lesson (e.g., "Linux", "Docker", "Kubernetes")
	Category string `json:"category" bson:"category"`

	// Tags is a list of tags associated with the lesson for filtering and search
	Tags []string `json:"tags" bson:"tags"`

	// Difficulty indicates the complexity level of the lesson (e.g., "Beginner", "Intermediate", "Advanced")
	Difficulty string `json:"difficulty" bson:"difficulty"`

	// EstimatedTime is the estimated time to complete the lesson in minutes
	EstimatedTime int `json:"estimated_time" bson:"estimated_time"`

	// DefaultImage is the default Docker image to use for all steps in the lesson
	// Individual steps can override this with their own Image field
	DefaultImage string `json:"default_image" bson:"default_image"`

	// Default resource limits for all steps in the lesson
	// Individual steps can override these with their own resource limit fields

	// DefaultMaxProcesses is the default maximum number of processes that can be created in the container
	DefaultMaxProcesses int64 `json:"default_max_processes,omitempty" bson:"default_max_processes,omitempty"`

	// DefaultMaxMemoryMB is the default maximum amount of memory that the container can use, in megabytes
	DefaultMaxMemoryMB int64 `json:"default_max_memory_mb,omitempty" bson:"default_max_memory_mb,omitempty"`

	// DefaultStorageSize is the default maximum amount of storage that the container can use
	DefaultStorageSize string `json:"default_storage_size,omitempty" bson:"default_storage_size,omitempty"`

	// Steps is an ordered list of steps that make up the lesson
	Steps []LessonStep `json:"steps" bson:"steps"`

	// CreatedAt is the timestamp when the lesson was created
	CreatedAt time.Time `json:"created_at" bson:"created_at"`

	// UpdatedAt is the timestamp when the lesson was last updated
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`

	// Version is the current version number of the lesson
	Version int `json:"version" bson:"version"`

	// VersionHistory contains information about previous versions of the lesson
	VersionHistory []VersionInfo `json:"version_history" bson:"version_history"`

	// CurrentStep is the index of the current step in the lesson
	CurrentStep int `json:"current_step" bson:"current_step"`
}
