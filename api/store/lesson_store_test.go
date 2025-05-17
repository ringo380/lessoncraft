package store

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ringo380/lessoncraft/lesson"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a test lesson
func createTestLesson() lesson.Lesson {
	return lesson.Lesson{
		ID:            uuid.New().String(),
		Title:         "Test Lesson",
		Description:   "This is a test lesson",
		Category:      "Test Category",
		Tags:          []string{"test", "example"},
		Difficulty:    "Beginner",
		EstimatedTime: 30,
		Steps: []lesson.LessonStep{
			{
				ID:       "step-1",
				Content:  "Step 1 content",
				Commands: []string{"echo 'Hello, World!'"},
				Expected: "Hello, World!",
				Timeout:  5 * time.Minute,
			},
		},
		CreatedAt:   time.Now(),
		CurrentStep: 0,
	}
}

// Test ListLessons
func TestListLessons(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lessons
	lesson1 := createTestLesson()
	lesson2 := createTestLesson()
	store.CreateLesson(&lesson1)
	store.CreateLesson(&lesson2)

	// Call the method being tested with default options
	result, err := store.ListLessons(DefaultListOptions())

	// Assert expectations
	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.TotalItems)
	assert.Equal(t, int64(1), result.TotalPages)
}

// Test ListAllLessons
func TestListAllLessons(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lessons
	lesson1 := createTestLesson()
	lesson2 := createTestLesson()
	store.CreateLesson(&lesson1)
	store.CreateLesson(&lesson2)

	// Call the method being tested
	lessons, err := store.ListAllLessons()

	// Assert expectations
	assert.NoError(t, err)
	assert.Len(t, lessons, 2)
}

// Test GetLesson
func TestGetLesson(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lesson
	testLesson := createTestLesson()
	store.CreateLesson(&testLesson)

	// Call the method being tested
	lesson, err := store.GetLesson(testLesson.ID)

	// Assert expectations
	assert.NoError(t, err)
	assert.Equal(t, testLesson.ID, lesson.ID)
	assert.Equal(t, testLesson.Title, lesson.Title)
}

// Test GetLesson with non-existent ID
func TestGetLessonNotFound(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Call the method being tested with a non-existent ID
	_, err := store.GetLesson("non-existent-id")

	// Assert expectations
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test CreateLesson
func TestCreateLesson(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create test lesson
	testLesson := createTestLesson()

	// Call the method being tested
	err := store.CreateLesson(&testLesson)

	// Assert expectations
	assert.NoError(t, err)

	// Verify the lesson was added to the store
	storedLesson, err := store.GetLesson(testLesson.ID)
	assert.NoError(t, err)
	assert.Equal(t, testLesson.ID, storedLesson.ID)
}

// Test CreateLesson with empty ID
func TestCreateLessonEmptyID(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create test lesson with empty ID
	testLesson := createTestLesson()
	testLesson.ID = ""

	// Call the method being tested
	err := store.CreateLesson(&testLesson)

	// Assert expectations
	assert.NoError(t, err)
	assert.NotEmpty(t, testLesson.ID)

	// Verify the lesson was added to the store
	storedLesson, err := store.GetLesson(testLesson.ID)
	assert.NoError(t, err)
	assert.Equal(t, testLesson.ID, storedLesson.ID)
}

// Test UpdateLesson
func TestUpdateLesson(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lesson
	testLesson := createTestLesson()
	store.CreateLesson(&testLesson)

	// Modify the lesson
	testLesson.Title = "Updated Title"

	// Call the method being tested
	err := store.UpdateLesson(testLesson.ID, &testLesson, "Updated title")

	// Assert expectations
	assert.NoError(t, err)

	// Verify the lesson was updated in the store
	storedLesson, err := store.GetLesson(testLesson.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", storedLesson.Title)
	assert.Equal(t, 2, storedLesson.Version)
	assert.Len(t, storedLesson.VersionHistory, 1)
	assert.Equal(t, 1, storedLesson.VersionHistory[0].Version)
	assert.Equal(t, "Updated title", storedLesson.VersionHistory[0].ChangeSummary)
}

// Test UpdateLesson with non-existent ID
func TestUpdateLessonNotFound(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create test lesson
	testLesson := createTestLesson()

	// Call the method being tested with a non-existent ID
	err := store.UpdateLesson("non-existent-id", &testLesson, "Update that should fail")

	// Assert expectations
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test DeleteLesson
func TestDeleteLesson(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lesson
	testLesson := createTestLesson()
	store.CreateLesson(&testLesson)

	// Call the method being tested
	err := store.DeleteLesson(testLesson.ID)

	// Assert expectations
	assert.NoError(t, err)

	// Verify the lesson was deleted from the store
	_, err = store.GetLesson(testLesson.ID)
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test DeleteLesson with non-existent ID
func TestDeleteLessonNotFound(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Call the method being tested with a non-existent ID
	err := store.DeleteLesson("non-existent-id")

	// Assert expectations
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test GetLessonVersion
func TestGetLessonVersion(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lesson
	testLesson := createTestLesson()
	store.CreateLesson(&testLesson)

	// Initial version should be 1
	assert.Equal(t, 1, testLesson.Version)

	// Update the lesson to create version 2
	testLesson.Title = "Updated Title"
	err := store.UpdateLesson(testLesson.ID, &testLesson, "Updated title")
	assert.NoError(t, err)

	// Update again to create version 3
	testLesson.Description = "Updated Description"
	err = store.UpdateLesson(testLesson.ID, &testLesson, "Updated description")
	assert.NoError(t, err)

	// Get the current version (version 3)
	currentLesson, err := store.GetLesson(testLesson.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, currentLesson.Version)
	assert.Equal(t, "Updated Title", currentLesson.Title)
	assert.Equal(t, "Updated Description", currentLesson.Description)

	// Get version 2
	v2Lesson, err := store.GetLessonVersion(testLesson.ID, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, v2Lesson.Version)
	assert.Equal(t, "Updated Title", v2Lesson.Title)
	assert.Equal(t, "", v2Lesson.Description) // Description was updated in version 3

	// Get version 1
	v1Lesson, err := store.GetLessonVersion(testLesson.ID, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, v1Lesson.Version)
	assert.Equal(t, "Test Lesson", v1Lesson.Title)                 // Original title
	assert.Equal(t, "This is a test lesson", v1Lesson.Description) // Original description

	// Try to get a non-existent version
	_, err = store.GetLessonVersion(testLesson.ID, 4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")

	// Try to get a lesson with a non-existent ID
	_, err = store.GetLessonVersion("non-existent-id", 1)
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test ListLessonVersions
func TestListLessonVersions(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lesson
	testLesson := createTestLesson()
	store.CreateLesson(&testLesson)

	// Update the lesson twice to create versions 2 and 3
	testLesson.Title = "Updated Title"
	err := store.UpdateLesson(testLesson.ID, &testLesson, "Updated title")
	assert.NoError(t, err)

	testLesson.Description = "Updated Description"
	err = store.UpdateLesson(testLesson.ID, &testLesson, "Updated description")
	assert.NoError(t, err)

	// List all versions
	versions, err := store.ListLessonVersions(testLesson.ID)
	assert.NoError(t, err)
	assert.Len(t, versions, 3) // Should have 3 versions

	// Versions should be sorted by version number in descending order
	assert.Equal(t, 3, versions[0].Version)
	assert.Equal(t, 2, versions[1].Version)
	assert.Equal(t, 1, versions[2].Version)

	// Check change summaries
	assert.Equal(t, "Current version", versions[0].ChangeSummary)
	assert.Equal(t, "Updated description", versions[1].ChangeSummary)
	assert.Equal(t, "Updated title", versions[2].ChangeSummary)

	// Try to list versions for a non-existent lesson
	_, err = store.ListLessonVersions("non-existent-id")
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test ListCategories
func TestListCategories(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lessons with different categories
	lesson1 := createTestLesson()
	lesson1.Category = "Category A"

	lesson2 := createTestLesson()
	lesson2.Category = "Category B"

	lesson3 := createTestLesson()
	lesson3.Category = "Category A" // Duplicate category

	store.CreateLesson(&lesson1)
	store.CreateLesson(&lesson2)
	store.CreateLesson(&lesson3)

	// Call the method being tested
	categories, err := store.ListCategories()

	// Assert expectations
	assert.NoError(t, err)
	assert.Len(t, categories, 2) // Should have 2 unique categories
	assert.Contains(t, categories, "Category A")
	assert.Contains(t, categories, "Category B")

	// Categories should be sorted alphabetically
	assert.Equal(t, "Category A", categories[0])
	assert.Equal(t, "Category B", categories[1])
}

// Test ListTags
func TestListTags(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lessons with different tags
	lesson1 := createTestLesson()
	lesson1.Tags = []string{"tag1", "tag2"}

	lesson2 := createTestLesson()
	lesson2.Tags = []string{"tag2", "tag3"}

	lesson3 := createTestLesson()
	lesson3.Tags = []string{"tag1", "tag4"}

	store.CreateLesson(&lesson1)
	store.CreateLesson(&lesson2)
	store.CreateLesson(&lesson3)

	// Call the method being tested
	tags, err := store.ListTags()

	// Assert expectations
	assert.NoError(t, err)
	assert.Len(t, tags, 4) // Should have 4 unique tags
	assert.Contains(t, tags, "tag1")
	assert.Contains(t, tags, "tag2")
	assert.Contains(t, tags, "tag3")
	assert.Contains(t, tags, "tag4")

	// Tags should be sorted alphabetically
	assert.Equal(t, "tag1", tags[0])
	assert.Equal(t, "tag2", tags[1])
	assert.Equal(t, "tag3", tags[2])
	assert.Equal(t, "tag4", tags[3])
}

// Test AddTag
func TestAddTag(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lesson
	testLesson := createTestLesson()
	testLesson.Tags = []string{"initial-tag"}
	store.CreateLesson(&testLesson)

	// Call the method being tested
	err := store.AddTag(testLesson.ID, "new-tag")
	assert.NoError(t, err)

	// Verify the tag was added
	storedLesson, err := store.GetLesson(testLesson.ID)
	assert.NoError(t, err)
	assert.Len(t, storedLesson.Tags, 2)
	assert.Contains(t, storedLesson.Tags, "initial-tag")
	assert.Contains(t, storedLesson.Tags, "new-tag")

	// Adding the same tag again should not duplicate it
	err = store.AddTag(testLesson.ID, "new-tag")
	assert.NoError(t, err)

	storedLesson, err = store.GetLesson(testLesson.ID)
	assert.NoError(t, err)
	assert.Len(t, storedLesson.Tags, 2) // Still only 2 tags

	// Test adding a tag to a non-existent lesson
	err = store.AddTag("non-existent-id", "tag")
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test RemoveTag
func TestRemoveTag(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lesson
	testLesson := createTestLesson()
	testLesson.Tags = []string{"tag1", "tag2", "tag3"}
	store.CreateLesson(&testLesson)

	// Call the method being tested
	err := store.RemoveTag(testLesson.ID, "tag2")
	assert.NoError(t, err)

	// Verify the tag was removed
	storedLesson, err := store.GetLesson(testLesson.ID)
	assert.NoError(t, err)
	assert.Len(t, storedLesson.Tags, 2)
	assert.Contains(t, storedLesson.Tags, "tag1")
	assert.Contains(t, storedLesson.Tags, "tag3")
	assert.NotContains(t, storedLesson.Tags, "tag2")

	// Removing a non-existent tag should not error
	err = store.RemoveTag(testLesson.ID, "non-existent-tag")
	assert.NoError(t, err)

	// Test removing a tag from a non-existent lesson
	err = store.RemoveTag("non-existent-id", "tag")
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test SetCategory
func TestSetCategory(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lesson
	testLesson := createTestLesson()
	testLesson.Category = "Initial Category"
	store.CreateLesson(&testLesson)

	// Call the method being tested
	err := store.SetCategory(testLesson.ID, "New Category")
	assert.NoError(t, err)

	// Verify the category was updated
	storedLesson, err := store.GetLesson(testLesson.ID)
	assert.NoError(t, err)
	assert.Equal(t, "New Category", storedLesson.Category)

	// Test setting a category for a non-existent lesson
	err = store.SetCategory("non-existent-id", "Category")
	assert.Error(t, err)
	assert.Equal(t, "lesson not found", err.Error())
}

// Test ListLessonsByCategory
func TestListLessonsByCategory(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lessons with different categories
	lesson1 := createTestLesson()
	lesson1.Category = "Category A"
	lesson1.Title = "Lesson 1"

	lesson2 := createTestLesson()
	lesson2.Category = "Category B"
	lesson2.Title = "Lesson 2"

	lesson3 := createTestLesson()
	lesson3.Category = "Category A"
	lesson3.Title = "Lesson 3"

	store.CreateLesson(&lesson1)
	store.CreateLesson(&lesson2)
	store.CreateLesson(&lesson3)

	// Call the method being tested
	result, err := store.ListLessonsByCategory("Category A", DefaultListOptions())

	// Assert expectations
	assert.NoError(t, err)
	assert.Len(t, result.Items, 2) // Should have 2 lessons in Category A
	assert.Equal(t, int64(2), result.TotalItems)

	// Verify the correct lessons were returned
	titles := []string{result.Items[0].Title, result.Items[1].Title}
	assert.Contains(t, titles, "Lesson 1")
	assert.Contains(t, titles, "Lesson 3")

	// Test with a category that has no lessons
	result, err = store.ListLessonsByCategory("Non-existent Category", DefaultListOptions())
	assert.NoError(t, err)
	assert.Len(t, result.Items, 0)
	assert.Equal(t, int64(0), result.TotalItems)
}

// Test ListLessonsByTag
func TestListLessonsByTag(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lessons with different tags
	lesson1 := createTestLesson()
	lesson1.Tags = []string{"tag1", "tag2"}
	lesson1.Title = "Lesson 1"

	lesson2 := createTestLesson()
	lesson2.Tags = []string{"tag2", "tag3"}
	lesson2.Title = "Lesson 2"

	lesson3 := createTestLesson()
	lesson3.Tags = []string{"tag1", "tag4"}
	lesson3.Title = "Lesson 3"

	store.CreateLesson(&lesson1)
	store.CreateLesson(&lesson2)
	store.CreateLesson(&lesson3)

	// Call the method being tested
	result, err := store.ListLessonsByTag("tag1", DefaultListOptions())

	// Assert expectations
	assert.NoError(t, err)
	assert.Len(t, result.Items, 2) // Should have 2 lessons with tag1
	assert.Equal(t, int64(2), result.TotalItems)

	// Verify the correct lessons were returned
	titles := []string{result.Items[0].Title, result.Items[1].Title}
	assert.Contains(t, titles, "Lesson 1")
	assert.Contains(t, titles, "Lesson 3")

	// Test with tag2 which should return 2 different lessons
	result, err = store.ListLessonsByTag("tag2", DefaultListOptions())
	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
	titles = []string{result.Items[0].Title, result.Items[1].Title}
	assert.Contains(t, titles, "Lesson 1")
	assert.Contains(t, titles, "Lesson 2")

	// Test with a tag that has no lessons
	result, err = store.ListLessonsByTag("non-existent-tag", DefaultListOptions())
	assert.NoError(t, err)
	assert.Len(t, result.Items, 0)
	assert.Equal(t, int64(0), result.TotalItems)
}

// Test SearchLessons
func TestSearchLessons(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create and add test lessons with different attributes
	lesson1 := createTestLesson()
	lesson1.Title = "Introduction to Docker"
	lesson1.Description = "Learn the basics of Docker containers"
	lesson1.Category = "Docker"
	lesson1.Tags = []string{"containers", "devops", "beginner"}
	lesson1.Difficulty = "Beginner"
	lesson1.EstimatedTime = 30
	lesson1.Steps[0].Content = "Docker is a platform for developing, shipping, and running applications in containers."

	lesson2 := createTestLesson()
	lesson2.Title = "Advanced Kubernetes"
	lesson2.Description = "Master Kubernetes orchestration"
	lesson2.Category = "Kubernetes"
	lesson2.Tags = []string{"containers", "devops", "advanced"}
	lesson2.Difficulty = "Advanced"
	lesson2.EstimatedTime = 120
	lesson2.Steps[0].Content = "Kubernetes is an open-source container orchestration platform."

	lesson3 := createTestLesson()
	lesson3.Title = "Docker Compose"
	lesson3.Description = "Learn how to use Docker Compose"
	lesson3.Category = "Docker"
	lesson3.Tags = []string{"containers", "devops", "intermediate"}
	lesson3.Difficulty = "Intermediate"
	lesson3.EstimatedTime = 60
	lesson3.Steps[0].Content = "Docker Compose is a tool for defining and running multi-container Docker applications."

	lesson4 := createTestLesson()
	lesson4.Title = "Linux Basics"
	lesson4.Description = "Introduction to Linux commands"
	lesson4.Category = "Linux"
	lesson4.Tags = []string{"linux", "command-line", "beginner"}
	lesson4.Difficulty = "Beginner"
	lesson4.EstimatedTime = 45
	lesson4.Steps[0].Content = "Linux is a family of open-source Unix-like operating systems."

	store.CreateLesson(&lesson1)
	store.CreateLesson(&lesson2)
	store.CreateLesson(&lesson3)
	store.CreateLesson(&lesson4)

	// Test cases
	tests := []struct {
		name           string
		searchOptions  SearchOptions
		expectedCount  int
		expectedTitles []string
	}{
		{
			name: "search by query in title",
			searchOptions: SearchOptions{
				Query: "Docker",
				Page:  1,
			},
			expectedCount:  2,
			expectedTitles: []string{"Docker Compose", "Introduction to Docker"},
		},
		{
			name: "search by query in description",
			searchOptions: SearchOptions{
				Query: "Kubernetes",
				Page:  1,
			},
			expectedCount:  1,
			expectedTitles: []string{"Advanced Kubernetes"},
		},
		{
			name: "search by query in content",
			searchOptions: SearchOptions{
				Query:          "container orchestration",
				IncludeContent: true,
				Page:           1,
			},
			expectedCount:  1,
			expectedTitles: []string{"Advanced Kubernetes"},
		},
		{
			name: "search by category",
			searchOptions: SearchOptions{
				Categories: []string{"Docker"},
				Page:       1,
			},
			expectedCount:  2,
			expectedTitles: []string{"Docker Compose", "Introduction to Docker"},
		},
		{
			name: "search by multiple categories",
			searchOptions: SearchOptions{
				Categories: []string{"Docker", "Linux"},
				Page:       1,
			},
			expectedCount:  3,
			expectedTitles: []string{"Docker Compose", "Introduction to Docker", "Linux Basics"},
		},
		{
			name: "search by tag",
			searchOptions: SearchOptions{
				Tags: []string{"beginner"},
				Page: 1,
			},
			expectedCount:  2,
			expectedTitles: []string{"Introduction to Docker", "Linux Basics"},
		},
		{
			name: "search by required tags",
			searchOptions: SearchOptions{
				RequiredTags: []string{"containers", "devops"},
				Page:         1,
			},
			expectedCount:  3,
			expectedTitles: []string{"Advanced Kubernetes", "Docker Compose", "Introduction to Docker"},
		},
		{
			name: "search by difficulty",
			searchOptions: SearchOptions{
				Difficulty: "Intermediate",
				Page:       1,
			},
			expectedCount:  1,
			expectedTitles: []string{"Docker Compose"},
		},
		{
			name: "search by min estimated time",
			searchOptions: SearchOptions{
				MinEstimatedTime: 60,
				Page:             1,
			},
			expectedCount:  2,
			expectedTitles: []string{"Advanced Kubernetes", "Docker Compose"},
		},
		{
			name: "search by max estimated time",
			searchOptions: SearchOptions{
				MaxEstimatedTime: 45,
				Page:             1,
			},
			expectedCount:  2,
			expectedTitles: []string{"Introduction to Docker", "Linux Basics"},
		},
		{
			name: "search by estimated time range",
			searchOptions: SearchOptions{
				MinEstimatedTime: 30,
				MaxEstimatedTime: 60,
				Page:             1,
			},
			expectedCount:  3,
			expectedTitles: []string{"Docker Compose", "Introduction to Docker", "Linux Basics"},
		},
		{
			name: "combined search",
			searchOptions: SearchOptions{
				Query:          "Docker",
				Categories:     []string{"Docker"},
				Tags:           []string{"beginner", "intermediate"},
				Difficulty:     "Beginner",
				IncludeContent: true,
				Page:           1,
			},
			expectedCount:  1,
			expectedTitles: []string{"Introduction to Docker"},
		},
		{
			name: "search with sorting",
			searchOptions: SearchOptions{
				Categories: []string{"Docker", "Kubernetes"},
				Sort:       map[string]int{"estimatedTime": 1}, // Sort by estimated time ascending
				Page:       1,
			},
			expectedCount:  3,
			expectedTitles: []string{"Introduction to Docker", "Docker Compose", "Advanced Kubernetes"},
		},
		{
			name: "search with pagination",
			searchOptions: SearchOptions{
				Tags:     []string{"containers", "devops"},
				Page:     1,
				PageSize: 2,
			},
			expectedCount:  2,                                                 // 2 items per page
			expectedTitles: []string{"Advanced Kubernetes", "Docker Compose"}, // First page
		},
		{
			name: "search with pagination - page 2",
			searchOptions: SearchOptions{
				Tags:     []string{"containers", "devops"},
				Page:     2,
				PageSize: 2,
			},
			expectedCount:  1,                                  // 1 item on second page
			expectedTitles: []string{"Introduction to Docker"}, // Second page
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the method being tested
			result, err := store.SearchLessons(tt.searchOptions)

			// Assert expectations
			assert.NoError(t, err)
			assert.Len(t, result.Items, tt.expectedCount)

			// Check that all expected titles are present
			titles := make([]string, len(result.Items))
			for i, lesson := range result.Items {
				titles[i] = lesson.Title
			}

			for _, expectedTitle := range tt.expectedTitles {
				assert.Contains(t, titles, expectedTitle)
			}

			// Check pagination metadata
			assert.Equal(t, tt.searchOptions.Page, result.Page)
			if tt.searchOptions.PageSize > 0 {
				assert.Equal(t, tt.searchOptions.PageSize, result.PageSize)
			} else {
				assert.Equal(t, int64(20), result.PageSize) // Default page size
			}
		})
	}
}

// Test concurrent operations
func TestConcurrentOperations(t *testing.T) {
	// Create store
	store := NewMemoryLessonStore()

	// Create a large number of lessons concurrently
	const numLessons = 100
	var wg sync.WaitGroup
	wg.Add(numLessons)

	for i := 0; i < numLessons; i++ {
		go func() {
			defer wg.Done()
			lesson := createTestLesson()
			err := store.CreateLesson(&lesson)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	// Verify all lessons were added
	result, err := store.ListLessons(DefaultListOptions())
	assert.NoError(t, err)
	assert.Equal(t, int64(numLessons), result.TotalItems)
}
