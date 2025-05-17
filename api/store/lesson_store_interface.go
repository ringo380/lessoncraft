package store

import (
	"github.com/ringo380/lessoncraft/lesson"
)

// SearchOptions defines the options for searching lessons
type SearchOptions struct {
	// Query is the main search term to match against lesson title, description, and content
	Query string

	// Categories is a list of categories to filter by (OR logic - lesson must be in at least one)
	Categories []string

	// Tags is a list of tags to filter by (OR logic - lesson must have at least one)
	Tags []string

	// RequiredTags is a list of tags that must all be present (AND logic)
	RequiredTags []string

	// Difficulty filters lessons by difficulty level (e.g., "Beginner", "Intermediate", "Advanced")
	Difficulty string

	// MaxEstimatedTime filters lessons by maximum estimated completion time in minutes
	MaxEstimatedTime int

	// MinEstimatedTime filters lessons by minimum estimated completion time in minutes
	MinEstimatedTime int

	// IncludeContent determines whether to search in lesson step content
	IncludeContent bool

	// Pagination options
	Page     int64
	PageSize int64

	// Sorting options (field name -> 1 for ascending, -1 for descending)
	Sort map[string]int
}

// SearchResult represents the result of a search operation
type SearchResult struct {
	// Items contains the lessons matching the search criteria for the current page
	Items []lesson.Lesson

	// TotalItems is the total number of lessons matching the search criteria across all pages
	TotalItems int64

	// TotalPages is the total number of pages of results
	TotalPages int64

	// Page is the current page number
	Page int64

	// PageSize is the number of items per page
	PageSize int64
}

// LessonStore defines the interface for lesson storage operations
type LessonStore interface {
	// ListLessons retrieves lessons with pagination
	ListLessons(opts ListOptions) (*ListResult, error)

	// ListAllLessons retrieves all lessons without pagination
	ListAllLessons() ([]lesson.Lesson, error)

	// GetLesson retrieves a lesson by ID
	GetLesson(id string) (*lesson.Lesson, error)

	// GetLessonVersion retrieves a specific version of a lesson
	GetLessonVersion(id string, version int) (*lesson.Lesson, error)

	// ListLessonVersions retrieves information about all versions of a lesson
	ListLessonVersions(id string) ([]lesson.VersionInfo, error)

	// CreateLesson adds a new lesson
	CreateLesson(l *lesson.Lesson) error

	// UpdateLesson updates an existing lesson
	UpdateLesson(id string, l *lesson.Lesson, changeSummary string) error

	// DeleteLesson removes a lesson
	DeleteLesson(id string) error

	// Category and Tag Operations

	// ListCategories retrieves all unique categories used in lessons
	ListCategories() ([]string, error)

	// ListTags retrieves all unique tags used in lessons
	ListTags() ([]string, error)

	// AddTag adds a tag to a lesson
	AddTag(id string, tag string) error

	// RemoveTag removes a tag from a lesson
	RemoveTag(id string, tag string) error

	// SetCategory sets the category for a lesson
	SetCategory(id string, category string) error

	// ListLessonsByCategory retrieves lessons in a specific category
	ListLessonsByCategory(category string, opts ListOptions) (*ListResult, error)

	// ListLessonsByTag retrieves lessons with a specific tag
	ListLessonsByTag(tag string, opts ListOptions) (*ListResult, error)

	// Search Operations

	// SearchLessons searches for lessons based on various criteria
	SearchLessons(opts SearchOptions) (*SearchResult, error)
}
