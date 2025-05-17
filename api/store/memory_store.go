package store

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ringo380/lessoncraft/lesson"
)

// MemoryLessonStore is an in-memory implementation of the LessonStore interface.
// It is primarily used for testing purposes, providing a simple and fast way to
// store and retrieve lessons without requiring a database connection.
// The implementation is thread-safe, using a read-write mutex to protect access
// to the underlying map of lessons.
type MemoryLessonStore struct {
	lessons map[string]*lesson.Lesson // Map of lesson ID to lesson pointer
	mu      sync.RWMutex              // Mutex to protect concurrent access
}

// NewMemoryLessonStore creates a new in-memory lesson store.
// It initializes an empty map to store lessons.
//
// Returns:
//   - A pointer to a new MemoryLessonStore
func NewMemoryLessonStore() *MemoryLessonStore {
	return &MemoryLessonStore{
		lessons: make(map[string]*lesson.Lesson),
	}
}

// ListLessons retrieves lessons from the in-memory store with pagination.
// It supports filtering, sorting, and pagination through the ListOptions parameter.
//
// Parameters:
//   - opts: Options for pagination, sorting, and filtering
//
// Returns:
//   - A ListResult containing the paginated results and metadata
//   - An error (always nil for this implementation)
func (s *MemoryLessonStore) ListLessons(opts ListOptions) (*ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert map to slice for easier manipulation
	allLessons := make([]lesson.Lesson, 0, len(s.lessons))
	for _, l := range s.lessons {
		allLessons = append(allLessons, *l)
	}

	// Apply filtering with support for categories and tags
	var filteredLessons []lesson.Lesson
	if len(opts.Filter) > 0 {
		for _, l := range allLessons {
			include := true
			for k, v := range opts.Filter {
				switch k {
				case "id":
					// Exact match for ID
					if l.ID != v {
						include = false
					}
				case "title":
					// Case-insensitive substring match for title
					if !strings.Contains(strings.ToLower(l.Title), strings.ToLower(v.(string))) {
						include = false
					}
				case "category":
					// Exact match for category
					if l.Category != v {
						include = false
					}
				case "difficulty":
					// Exact match for difficulty
					if l.Difficulty != v {
						include = false
					}
				case "tag":
					// Check if the lesson has the specified tag
					tagFound := false
					for _, tag := range l.Tags {
						if tag == v {
							tagFound = true
							break
						}
					}
					if !tagFound {
						include = false
					}
				case "tags":
					// Check if the lesson has all the specified tags
					if tags, ok := v.([]string); ok {
						for _, requiredTag := range tags {
							tagFound := false
							for _, lessonTag := range l.Tags {
								if lessonTag == requiredTag {
									tagFound = true
									break
								}
							}
							if !tagFound {
								include = false
								break
							}
						}
					}
				case "estimatedTime":
					// Filter by estimated time (less than or equal)
					if time, ok := v.(int); ok {
						if l.EstimatedTime > time {
							include = false
						}
					}
				}

				if !include {
					break
				}
			}
			if include {
				filteredLessons = append(filteredLessons, l)
			}
		}
	} else {
		filteredLessons = allLessons
	}

	// Apply sorting (simple implementation that only sorts by createdAt)
	if len(opts.Sort) > 0 {
		sort.Slice(filteredLessons, func(i, j int) bool {
			for k, v := range opts.Sort {
				if k == "createdAt" {
					if v == 1 {
						return filteredLessons[i].CreatedAt.Before(filteredLessons[j].CreatedAt)
					} else {
						return filteredLessons[i].CreatedAt.After(filteredLessons[j].CreatedAt)
					}
				} else if k == "title" {
					if v == 1 {
						return filteredLessons[i].Title < filteredLessons[j].Title
					} else {
						return filteredLessons[i].Title > filteredLessons[j].Title
					}
				}
			}
			return false
		})
	}

	// Calculate pagination
	totalItems := int64(len(filteredLessons))
	totalPages := totalItems / opts.PageSize
	if totalItems%opts.PageSize > 0 {
		totalPages++
	}

	// Apply pagination
	start := (opts.Page - 1) * opts.PageSize
	end := start + opts.PageSize
	if start >= int64(len(filteredLessons)) {
		start = 0
		end = 0
	}
	if end > int64(len(filteredLessons)) {
		end = int64(len(filteredLessons))
	}

	var paginatedLessons []lesson.Lesson
	if start < end {
		paginatedLessons = filteredLessons[start:end]
	} else {
		paginatedLessons = []lesson.Lesson{}
	}

	return &ListResult{
		Items:      paginatedLessons,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
	}, nil
}

// ListAllLessons retrieves all lessons from the in-memory store without pagination.
// It returns a copy of each lesson to prevent modification of the stored lessons.
//
// Returns:
//   - A slice of lesson.Lesson objects
//   - An error (always nil for this implementation)
func (s *MemoryLessonStore) ListAllLessons() ([]lesson.Lesson, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lessons := make([]lesson.Lesson, 0, len(s.lessons))
	for _, l := range s.lessons {
		lessons = append(lessons, *l)
	}
	return lessons, nil
}

// GetLesson retrieves a lesson by its ID from the in-memory store.
// It returns a pointer to the stored lesson, allowing for modification.
//
// Parameters:
//   - id: The ID of the lesson to retrieve
//
// Returns:
//   - A pointer to the retrieved lesson.Lesson object
//   - An error if the lesson is not found
func (s *MemoryLessonStore) GetLesson(id string) (*lesson.Lesson, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, ok := s.lessons[id]
	if !ok {
		return nil, errors.New("lesson not found")
	}
	return l, nil
}

// CreateLesson adds a new lesson to the in-memory store.
// If the lesson does not have an ID, it generates a new UUID.
// It initializes version-related fields and category/tag fields if not provided.
//
// Parameters:
//   - l: A pointer to the lesson.Lesson object to create
//
// Returns:
//   - An error (always nil for this implementation)
func (s *MemoryLessonStore) CreateLesson(l *lesson.Lesson) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if l.ID == "" {
		l.ID = uuid.New().String()
	}

	// Initialize version-related fields
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.UpdatedAt = now
	l.Version = 1
	l.VersionHistory = []lesson.VersionInfo{} // Initialize empty version history

	// Initialize category and tags if not provided
	if l.Category == "" {
		l.Category = "Uncategorized" // Default category
	}

	if l.Tags == nil {
		l.Tags = []string{} // Initialize empty tags slice
	}

	// Set default difficulty if not provided
	if l.Difficulty == "" {
		l.Difficulty = "Beginner" // Default difficulty
	}

	// Set default estimated time if not provided
	if l.EstimatedTime <= 0 {
		l.EstimatedTime = 30 // Default 30 minutes
	}

	s.lessons[l.ID] = l
	return nil
}

// UpdateLesson updates an existing lesson in the in-memory store.
// It handles versioning by incrementing the version number, updating the timestamp,
// and adding the previous version to the version history.
//
// Parameters:
//   - id: The ID of the lesson to update
//   - l: A pointer to the lesson.Lesson object with updated values
//   - changeSummary: A description of the changes made in this update
//
// Returns:
//   - An error if the lesson is not found
func (s *MemoryLessonStore) UpdateLesson(id string, l *lesson.Lesson, changeSummary string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentLesson, ok := s.lessons[id]
	if !ok {
		return errors.New("lesson not found")
	}

	// Create a version info record for the current version
	versionInfo := lesson.VersionInfo{
		Version:       currentLesson.Version,
		Timestamp:     currentLesson.UpdatedAt,
		ChangeSummary: changeSummary,
	}

	// Update version-related fields
	l.UpdatedAt = time.Now()
	l.Version = currentLesson.Version + 1

	// Append the current version to the version history
	l.VersionHistory = append(currentLesson.VersionHistory, versionInfo)

	// Update the lesson in the store
	s.lessons[id] = l
	return nil
}

// GetLessonVersion retrieves a specific version of a lesson from the in-memory store.
// If the requested version is the current version, it returns the lesson as is.
// If the requested version is in the version history, it reconstructs the lesson at that version.
//
// Parameters:
//   - id: The ID of the lesson to retrieve
//   - version: The version number to retrieve
//
// Returns:
//   - A pointer to the retrieved lesson.Lesson object at the specified version
//   - An error if the lesson is not found or the version doesn't exist
func (s *MemoryLessonStore) GetLessonVersion(id string, version int) (*lesson.Lesson, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get the current lesson
	currentLesson, ok := s.lessons[id]
	if !ok {
		return nil, errors.New("lesson not found")
	}

	// If the requested version is the current version, return the lesson as is
	if currentLesson.Version == version {
		return currentLesson, nil
	}

	// If the requested version is 0 or negative, return an error
	if version <= 0 {
		return nil, fmt.Errorf("invalid version number: %d", version)
	}

	// If the requested version is greater than the current version, return an error
	if version > currentLesson.Version {
		return nil, fmt.Errorf("version %d does not exist (current version is %d)", version, currentLesson.Version)
	}

	// Look for the requested version in the version history
	var versionInfo *lesson.VersionInfo
	for i := len(currentLesson.VersionHistory) - 1; i >= 0; i-- {
		if currentLesson.VersionHistory[i].Version == version {
			versionInfo = &currentLesson.VersionHistory[i]
			break
		}
	}

	// If the version wasn't found in the history, return an error
	if versionInfo == nil {
		return nil, fmt.Errorf("version %d not found in version history", version)
	}

	// For now, we don't have a way to reconstruct the exact state of a lesson at a previous version
	// This would require storing snapshots of each version or implementing a more complex versioning system
	// As a simple implementation, we'll return the current lesson but with the version and timestamp updated
	versionedLesson := *currentLesson
	versionedLesson.Version = version
	versionedLesson.UpdatedAt = versionInfo.Timestamp

	// Remove version history entries that came after the requested version
	var filteredHistory []lesson.VersionInfo
	for _, vi := range currentLesson.VersionHistory {
		if vi.Version < version {
			filteredHistory = append(filteredHistory, vi)
		}
	}
	versionedLesson.VersionHistory = filteredHistory

	return &versionedLesson, nil
}

// ListLessonVersions retrieves information about all versions of a lesson from the in-memory store.
// It returns a list of VersionInfo objects, including the current version and all previous versions.
// The list is sorted by version number in descending order (newest first).
//
// Parameters:
//   - id: The ID of the lesson to retrieve versions for
//
// Returns:
//   - A slice of lesson.VersionInfo objects representing all versions of the lesson
//   - An error if the lesson is not found
func (s *MemoryLessonStore) ListLessonVersions(id string) ([]lesson.VersionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get the current lesson
	currentLesson, ok := s.lessons[id]
	if !ok {
		return nil, errors.New("lesson not found")
	}

	// Create a list that includes both the current version and all versions in the history
	versions := make([]lesson.VersionInfo, 0, len(currentLesson.VersionHistory)+1)

	// Add the current version
	currentVersionInfo := lesson.VersionInfo{
		Version:       currentLesson.Version,
		Timestamp:     currentLesson.UpdatedAt,
		ChangeSummary: "Current version", // We don't have a change summary for the current version
	}
	versions = append(versions, currentVersionInfo)

	// Add all versions from the history
	versions = append(versions, currentLesson.VersionHistory...)

	// Sort the versions by version number in descending order (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	return versions, nil
}

// DeleteLesson removes a lesson from the in-memory store.
//
// Parameters:
//   - id: The ID of the lesson to delete
//
// Returns:
//   - An error if the lesson is not found
func (s *MemoryLessonStore) DeleteLesson(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.lessons[id]; !ok {
		return errors.New("lesson not found")
	}
	delete(s.lessons, id)
	return nil
}

// ListCategories retrieves all unique categories used in lessons.
// It returns a sorted list of category names.
//
// Returns:
//   - A slice of strings representing unique categories
//   - An error (always nil for this implementation)
func (s *MemoryLessonStore) ListCategories() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Use a map to track unique categories
	categoriesMap := make(map[string]bool)

	for _, l := range s.lessons {
		if l.Category != "" {
			categoriesMap[l.Category] = true
		}
	}

	// Convert map keys to slice
	categories := make([]string, 0, len(categoriesMap))
	for category := range categoriesMap {
		categories = append(categories, category)
	}

	// Sort categories alphabetically
	sort.Strings(categories)

	return categories, nil
}

// ListTags retrieves all unique tags used in lessons.
// It returns a sorted list of tag names.
//
// Returns:
//   - A slice of strings representing unique tags
//   - An error (always nil for this implementation)
func (s *MemoryLessonStore) ListTags() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Use a map to track unique tags
	tagsMap := make(map[string]bool)

	for _, l := range s.lessons {
		for _, tag := range l.Tags {
			if tag != "" {
				tagsMap[tag] = true
			}
		}
	}

	// Convert map keys to slice
	tags := make([]string, 0, len(tagsMap))
	for tag := range tagsMap {
		tags = append(tags, tag)
	}

	// Sort tags alphabetically
	sort.Strings(tags)

	return tags, nil
}

// AddTag adds a tag to a lesson.
// If the tag already exists on the lesson, it does nothing.
//
// Parameters:
//   - id: The ID of the lesson to modify
//   - tag: The tag to add
//
// Returns:
//   - An error if the lesson is not found
func (s *MemoryLessonStore) AddTag(id string, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lesson, ok := s.lessons[id]
	if !ok {
		return errors.New("lesson not found")
	}

	// Check if tag already exists
	for _, existingTag := range lesson.Tags {
		if existingTag == tag {
			return nil // Tag already exists, nothing to do
		}
	}

	// Add the tag
	lesson.Tags = append(lesson.Tags, tag)
	lesson.UpdatedAt = time.Now()

	return nil
}

// RemoveTag removes a tag from a lesson.
// If the tag doesn't exist on the lesson, it does nothing.
//
// Parameters:
//   - id: The ID of the lesson to modify
//   - tag: The tag to remove
//
// Returns:
//   - An error if the lesson is not found
func (s *MemoryLessonStore) RemoveTag(id string, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lesson, ok := s.lessons[id]
	if !ok {
		return errors.New("lesson not found")
	}

	// Find and remove the tag
	for i, existingTag := range lesson.Tags {
		if existingTag == tag {
			// Remove the tag by replacing it with the last element and truncating
			lesson.Tags[i] = lesson.Tags[len(lesson.Tags)-1]
			lesson.Tags = lesson.Tags[:len(lesson.Tags)-1]
			lesson.UpdatedAt = time.Now()
			break
		}
	}

	return nil
}

// SetCategory sets the category for a lesson.
//
// Parameters:
//   - id: The ID of the lesson to modify
//   - category: The new category
//
// Returns:
//   - An error if the lesson is not found
func (s *MemoryLessonStore) SetCategory(id string, category string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lesson, ok := s.lessons[id]
	if !ok {
		return errors.New("lesson not found")
	}

	// Set the category
	lesson.Category = category
	lesson.UpdatedAt = time.Now()

	return nil
}

// ListLessonsByCategory retrieves lessons in a specific category with pagination.
//
// Parameters:
//   - category: The category to filter by
//   - opts: Options for pagination, sorting, and additional filtering
//
// Returns:
//   - A ListResult containing the paginated results and metadata
//   - An error (always nil for this implementation)
func (s *MemoryLessonStore) ListLessonsByCategory(category string, opts ListOptions) (*ListResult, error) {
	// Add category filter to the options
	if opts.Filter == nil {
		opts.Filter = make(map[string]interface{})
	}
	opts.Filter["category"] = category

	// Use the existing ListLessons method
	return s.ListLessons(opts)
}

// ListLessonsByTag retrieves lessons with a specific tag with pagination.
//
// Parameters:
//   - tag: The tag to filter by
//   - opts: Options for pagination, sorting, and additional filtering
//
// Returns:
//   - A ListResult containing the paginated results and metadata
//   - An error (always nil for this implementation)
func (s *MemoryLessonStore) ListLessonsByTag(tag string, opts ListOptions) (*ListResult, error) {
	// Add tag filter to the options
	if opts.Filter == nil {
		opts.Filter = make(map[string]interface{})
	}
	opts.Filter["tag"] = tag

	// Use the existing ListLessons method
	return s.ListLessons(opts)
}

// SearchLessons searches for lessons based on various criteria.
// It supports searching by query text, categories, tags, difficulty, and estimated time.
// The search is performed on lesson title, description, and optionally on step content.
//
// Parameters:
//   - opts: Search options including query, filters, pagination, and sorting
//
// Returns:
//   - A SearchResult containing the search results and metadata
//   - An error (always nil for this implementation)
func (s *MemoryLessonStore) SearchLessons(opts SearchOptions) (*SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert map to slice for easier manipulation
	allLessons := make([]lesson.Lesson, 0, len(s.lessons))
	for _, l := range s.lessons {
		allLessons = append(allLessons, *l)
	}

	// Apply search criteria
	var matchedLessons []lesson.Lesson
	for _, l := range allLessons {
		// Start with the assumption that this lesson matches
		matches := true

		// Check query text (case-insensitive)
		if opts.Query != "" {
			queryLower := strings.ToLower(opts.Query)
			titleMatches := strings.Contains(strings.ToLower(l.Title), queryLower)
			descMatches := strings.Contains(strings.ToLower(l.Description), queryLower)

			// Check step content if requested
			contentMatches := false
			if opts.IncludeContent {
				for _, step := range l.Steps {
					if strings.Contains(strings.ToLower(step.Content), queryLower) {
						contentMatches = true
						break
					}
				}
			}

			// Lesson matches if query is found in title, description, or content (if included)
			if !(titleMatches || descMatches || contentMatches) {
				matches = false
			}
		}

		// Check categories (OR logic - lesson must be in at least one of the specified categories)
		if len(opts.Categories) > 0 {
			categoryMatches := false
			for _, category := range opts.Categories {
				if l.Category == category {
					categoryMatches = true
					break
				}
			}
			if !categoryMatches {
				matches = false
			}
		}

		// Check tags (OR logic - lesson must have at least one of the specified tags)
		if len(opts.Tags) > 0 {
			tagMatches := false
			for _, tag := range opts.Tags {
				for _, lessonTag := range l.Tags {
					if lessonTag == tag {
						tagMatches = true
						break
					}
				}
				if tagMatches {
					break
				}
			}
			if !tagMatches {
				matches = false
			}
		}

		// Check required tags (AND logic - lesson must have all specified tags)
		if len(opts.RequiredTags) > 0 {
			for _, requiredTag := range opts.RequiredTags {
				tagFound := false
				for _, lessonTag := range l.Tags {
					if lessonTag == requiredTag {
						tagFound = true
						break
					}
				}
				if !tagFound {
					matches = false
					break
				}
			}
		}

		// Check difficulty
		if opts.Difficulty != "" && l.Difficulty != opts.Difficulty {
			matches = false
		}

		// Check estimated time range
		if opts.MinEstimatedTime > 0 && l.EstimatedTime < opts.MinEstimatedTime {
			matches = false
		}
		if opts.MaxEstimatedTime > 0 && l.EstimatedTime > opts.MaxEstimatedTime {
			matches = false
		}

		// If all criteria match, include this lesson in the results
		if matches {
			matchedLessons = append(matchedLessons, l)
		}
	}

	// Apply sorting
	if len(opts.Sort) > 0 {
		sort.Slice(matchedLessons, func(i, j int) bool {
			for k, v := range opts.Sort {
				switch k {
				case "createdAt":
					if v == 1 {
						return matchedLessons[i].CreatedAt.Before(matchedLessons[j].CreatedAt)
					} else {
						return matchedLessons[i].CreatedAt.After(matchedLessons[j].CreatedAt)
					}
				case "updatedAt":
					if v == 1 {
						return matchedLessons[i].UpdatedAt.Before(matchedLessons[j].UpdatedAt)
					} else {
						return matchedLessons[i].UpdatedAt.After(matchedLessons[j].UpdatedAt)
					}
				case "title":
					if v == 1 {
						return matchedLessons[i].Title < matchedLessons[j].Title
					} else {
						return matchedLessons[i].Title > matchedLessons[j].Title
					}
				case "estimatedTime":
					if v == 1 {
						return matchedLessons[i].EstimatedTime < matchedLessons[j].EstimatedTime
					} else {
						return matchedLessons[i].EstimatedTime > matchedLessons[j].EstimatedTime
					}
				}
			}
			return false
		})
	} else {
		// Default sort by relevance (for now, just sort by title)
		sort.Slice(matchedLessons, func(i, j int) bool {
			return matchedLessons[i].Title < matchedLessons[j].Title
		})
	}

	// Calculate pagination
	totalItems := int64(len(matchedLessons))

	// Use default pagination if not specified
	page := opts.Page
	if page < 1 {
		page = 1
	}

	pageSize := opts.PageSize
	if pageSize < 1 {
		pageSize = 20 // Default page size
	}

	totalPages := totalItems / pageSize
	if totalItems%pageSize > 0 {
		totalPages++
	}

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= int64(len(matchedLessons)) {
		start = 0
		end = 0
	}
	if end > int64(len(matchedLessons)) {
		end = int64(len(matchedLessons))
	}

	var paginatedLessons []lesson.Lesson
	if start < end {
		paginatedLessons = matchedLessons[start:end]
	} else {
		paginatedLessons = []lesson.Lesson{}
	}

	return &SearchResult{
		Items:      paginatedLessons,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}
