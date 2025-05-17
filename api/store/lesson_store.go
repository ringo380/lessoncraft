package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/ringo380/lessoncraft/internal/circuitbreaker"
	"github.com/ringo380/lessoncraft/lesson"
	"log"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoLessonStore is an implementation of the LessonStore interface that uses MongoDB for storage.
// It provides methods for creating, retrieving, updating, and deleting lessons in a MongoDB database.
// The implementation includes retry logic with exponential backoff and circuit breaker for handling transient MongoDB errors.
type MongoLessonStore struct {
	db          *mongo.Database                // MongoDB database connection
	maxRetries  int                            // Maximum number of retry attempts
	baseBackoff time.Duration                  // Base duration for exponential backoff
	cb          *circuitbreaker.CircuitBreaker // Circuit breaker for MongoDB operations
}

// NewMongoLessonStore creates a new MongoLessonStore with the provided MongoDB database.
// It initializes the store with default retry settings (3 retries with 100ms base backoff)
// and ensures that the necessary indexes are created for optimal query performance.
//
// Parameters:
//   - db: A pointer to a MongoDB database connection
//
// Returns:
//   - A pointer to a new MongoLessonStore
func NewMongoLessonStore(db *mongo.Database) *MongoLessonStore {
	// Create a circuit breaker for MongoDB operations
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Options{
		Name:                     "mongodb",
		FailureThreshold:         5,
		ResetTimeout:             10 * time.Second,
		HalfOpenSuccessThreshold: 2,
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			log.Printf("MongoDB circuit breaker state changed from %v to %v", from, to)
		},
	})

	store := &MongoLessonStore{
		db:          db,
		maxRetries:  3,
		baseBackoff: 100 * time.Millisecond,
		cb:          cb,
	}

	// Ensure indexes are created
	if err := store.ensureIndexes(); err != nil {
		log.Printf("Warning: Failed to create indexes: %v", err)
	}

	return store
}

// ensureIndexes creates the necessary indexes on the lessons collection
// to optimize query performance. This includes indexes for frequently queried
// fields such as id, title, and createdAt.
//
// Returns:
//   - An error if the operation fails
func (s *MongoLessonStore) ensureIndexes() error {
	return s.withRetry("EnsureIndexes", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Collection reference
		collection := s.db.Collection("lessons")

		// Define indexes
		indexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{"id", 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys:    bson.D{{"title", 1}},
				Options: options.Index().SetBackground(true),
			},
			{
				Keys:    bson.D{{"createdAt", -1}},
				Options: options.Index().SetBackground(true),
			},
			{
				Keys:    bson.D{{"tags", 1}},
				Options: options.Index().SetBackground(true),
			},
		}

		// Create indexes
		_, err := collection.Indexes().CreateMany(ctx, indexes)
		return err
	})
}

// withRetry executes the given operation with retries, exponential backoff, and circuit breaker protection.
// If the operation fails with a retryable error, it will be retried up to maxRetries times
// with exponential backoff and jitter to avoid thundering herd problems.
// The circuit breaker will prevent repeated attempts to access a failing service after a threshold of failures.
//
// Parameters:
//   - operation: A string identifying the operation for logging purposes
//   - f: A function that performs the operation and returns an error if it fails
//
// Returns:
//   - The error from the last attempt, or nil if the operation succeeded
//   - circuitbreaker.ErrCircuitOpen if the circuit breaker is open
func (s *MongoLessonStore) withRetry(operation string, f func() error) error {
	// Use the circuit breaker to protect against repeated failures
	err := s.cb.Execute(func() error {
		var err error
		for attempt := 0; attempt <= s.maxRetries; attempt++ {
			if attempt > 0 {
				// Calculate backoff with jitter
				backoff := float64(s.baseBackoff) * math.Pow(2, float64(attempt-1))
				jitter := (rand.Float64() * 0.5) + 0.75 // 75% to 125% of backoff
				sleepTime := time.Duration(backoff * jitter)

				log.Printf("Retrying %s operation (attempt %d/%d) after %v due to: %v",
					operation, attempt, s.maxRetries, sleepTime, err)

				time.Sleep(sleepTime)
			}

			// Execute the operation
			err = f()

			// If successful or non-retryable error, return immediately
			if err == nil || !isRetryableError(err) {
				return err
			}
		}

		log.Printf("Failed %s operation after %d attempts: %v", operation, s.maxRetries+1, err)
		return err
	})

	// If the circuit is open, return a more descriptive error
	if err == circuitbreaker.ErrCircuitOpen {
		log.Printf("MongoDB circuit breaker is open for operation %s, too many failures detected", operation)
		return fmt.Errorf("MongoDB circuit breaker is open for operation %s: %w", operation, err)
	}

	return err
}

// isRetryableError determines if an error should trigger a retry operation.
// It checks for various types of transient errors that might be resolved by retrying,
// such as network errors, timeouts, and certain MongoDB-specific errors.
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - true if the error is retryable, false otherwise
func isRetryableError(err error) bool {
	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check for connection errors
	if mongo.IsNetworkError(err) {
		return true
	}

	// Check for server selection errors (by checking error message)
	if err != nil && (errors.Is(err, mongo.ErrClientDisconnected) ||
		errors.Is(err, mongo.ErrNoDocuments) ||
		errors.Is(err, mongo.ErrNilDocument) ||
		errors.Is(err, mongo.ErrNilValue) ||
		errors.Is(err, mongo.ErrEmptySlice)) {
		return true
	}

	// Add other retryable error types as needed

	return false
}

// ListOptions defines the options for listing lessons
type ListOptions struct {
	Page     int64                  // Page number (1-based)
	PageSize int64                  // Number of items per page
	Sort     map[string]int         // Sorting criteria (field name -> 1 for ascending, -1 for descending)
	Filter   map[string]interface{} // Filtering criteria
}

// DefaultListOptions returns the default options for listing lessons
func DefaultListOptions() ListOptions {
	return ListOptions{
		Page:     1,
		PageSize: 20,
		Sort:     map[string]int{"createdAt": -1},
		Filter:   map[string]interface{}{},
	}
}

// ListResult represents the result of a paginated list operation
type ListResult struct {
	Items      []lesson.Lesson // The items for the current page
	TotalItems int64           // Total number of items across all pages
	TotalPages int64           // Total number of pages
	Page       int64           // Current page number
	PageSize   int64           // Number of items per page
}

// ListLessons retrieves lessons from the MongoDB database with pagination.
// It uses the withRetry method to handle transient errors.
//
// Parameters:
//   - opts: Options for pagination, sorting, and filtering
//
// Returns:
//   - A ListResult containing the paginated results and metadata
//   - An error if the operation fails
func (s *MongoLessonStore) ListLessons(opts ListOptions) (*ListResult, error) {
	var result ListResult

	// Use default options if not specified
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 20
	}

	// Calculate skip value for pagination
	skip := (opts.Page - 1) * opts.PageSize

	err := s.withRetry("ListLessons", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create filter
		filter := bson.M{}
		for k, v := range opts.Filter {
			filter[k] = v
		}

		// Create sort specification
		sortBson := bson.D{}
		for k, v := range opts.Sort {
			sortBson = append(sortBson, bson.E{Key: k, Value: v})
		}

		// Count total documents for pagination metadata
		totalItems, err := s.db.Collection("lessons").CountDocuments(ctx, filter)
		if err != nil {
			return err
		}

		// Configure find options
		findOptions := options.Find().
			SetSkip(skip).
			SetLimit(opts.PageSize)

		if len(sortBson) > 0 {
			findOptions.SetSort(sortBson)
		}

		// Execute query
		cursor, err := s.db.Collection("lessons").Find(ctx, filter, findOptions)
		if err != nil {
			return err
		}
		defer cursor.Close(ctx)

		// Decode results
		var lessons []lesson.Lesson
		if err = cursor.All(ctx, &lessons); err != nil {
			return err
		}

		// Calculate pagination metadata
		totalPages := totalItems / opts.PageSize
		if totalItems%opts.PageSize > 0 {
			totalPages++
		}

		// Populate result
		result = ListResult{
			Items:      lessons,
			TotalItems: totalItems,
			TotalPages: totalPages,
			Page:       opts.Page,
			PageSize:   opts.PageSize,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

// ListAllLessons retrieves all lessons from the MongoDB database without pagination.
// This method should be used with caution for large collections.
// It uses the withRetry method to handle transient errors.
//
// Returns:
//   - A slice of lesson.Lesson objects
//   - An error if the operation fails
func (s *MongoLessonStore) ListAllLessons() ([]lesson.Lesson, error) {
	var lessons []lesson.Lesson

	err := s.withRetry("ListAllLessons", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := s.db.Collection("lessons").Find(ctx, bson.M{})
		if err != nil {
			return err
		}
		defer cursor.Close(ctx)

		if err = cursor.All(ctx, &lessons); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return lessons, nil
}

// GetLesson retrieves a lesson by its ID from the MongoDB database.
// It uses the withRetry method to handle transient errors.
//
// Parameters:
//   - id: The ID of the lesson to retrieve
//
// Returns:
//   - A pointer to the retrieved lesson.Lesson object
//   - An error if the operation fails or the lesson is not found
func (s *MongoLessonStore) GetLesson(id string) (*lesson.Lesson, error) {
	var lessonData lesson.Lesson

	err := s.withRetry("GetLesson", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := s.db.Collection("lessons").FindOne(ctx, bson.M{"id": id}).Decode(&lessonData)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &lessonData, nil
}

// GetLessonVersion retrieves a specific version of a lesson from the MongoDB database.
// If the requested version is the current version, it returns the lesson as is.
// If the requested version is in the version history, it reconstructs the lesson at that version.
// It uses the withRetry method to handle transient errors.
//
// Parameters:
//   - id: The ID of the lesson to retrieve
//   - version: The version number to retrieve
//
// Returns:
//   - A pointer to the retrieved lesson.Lesson object at the specified version
//   - An error if the operation fails, the lesson is not found, or the version doesn't exist
func (s *MongoLessonStore) GetLessonVersion(id string, version int) (*lesson.Lesson, error) {
	// First, get the current lesson
	currentLesson, err := s.GetLesson(id)
	if err != nil {
		return nil, err
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

// ListLessonVersions retrieves information about all versions of a lesson from the MongoDB database.
// It returns a list of VersionInfo objects, including the current version and all previous versions.
// The list is sorted by version number in descending order (newest first).
// It uses the withRetry method to handle transient errors.
//
// Parameters:
//   - id: The ID of the lesson to retrieve versions for
//
// Returns:
//   - A slice of lesson.VersionInfo objects representing all versions of the lesson
//   - An error if the operation fails or the lesson is not found
func (s *MongoLessonStore) ListLessonVersions(id string) ([]lesson.VersionInfo, error) {
	// First, get the current lesson
	currentLesson, err := s.GetLesson(id)
	if err != nil {
		return nil, err
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

// CreateLesson adds a new lesson to the MongoDB database.
// It generates a new UUID for the lesson, sets the creation time, and initializes version to 1.
// It uses the withRetry method to handle transient errors.
//
// Parameters:
//   - l: A pointer to the lesson.Lesson object to create
//
// Returns:
//   - An error if the operation fails
func (s *MongoLessonStore) CreateLesson(l *lesson.Lesson) error {
	// Set ID, creation time, and version before retries to ensure consistency
	l.ID = uuid.New().String()
	now := time.Now()
	l.CreatedAt = now
	l.UpdatedAt = now
	l.Version = 1
	l.VersionHistory = []lesson.VersionInfo{} // Initialize empty version history

	return s.withRetry("CreateLesson", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := s.db.Collection("lessons").InsertOne(ctx, l)
		return err
	})
}

// UpdateLesson updates an existing lesson in the MongoDB database.
// It handles versioning by incrementing the version number, updating the timestamp,
// and adding the previous version to the version history.
// It uses the withRetry method to handle transient errors.
//
// Parameters:
//   - id: The ID of the lesson to update
//   - l: A pointer to the lesson.Lesson object with updated values
//   - changeSummary: A description of the changes made in this update
//
// Returns:
//   - An error if the operation fails
func (s *MongoLessonStore) UpdateLesson(id string, l *lesson.Lesson, changeSummary string) error {
	return s.withRetry("UpdateLesson", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// First, get the current lesson to access its version information
		var currentLesson lesson.Lesson
		err := s.db.Collection("lessons").FindOne(ctx, bson.M{"id": id}).Decode(&currentLesson)
		if err != nil {
			return fmt.Errorf("failed to retrieve current lesson for versioning: %w", err)
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

		// Update the lesson in the database
		_, err = s.db.Collection("lessons").UpdateOne(
			ctx,
			bson.M{"id": id},
			bson.M{"$set": l},
		)
		return err
	})
}

// DeleteLesson removes a lesson from the MongoDB database.
// It uses the withRetry method to handle transient errors.
//
// Parameters:
//   - id: The ID of the lesson to delete
//
// Returns:
//   - An error if the operation fails
func (s *MongoLessonStore) DeleteLesson(id string) error {
	return s.withRetry("DeleteLesson", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := s.db.Collection("lessons").DeleteOne(ctx, bson.M{"id": id})
		return err
	})
}

// SearchLessons searches for lessons in the MongoDB database based on various criteria.
// It constructs a MongoDB query based on the search options and applies sorting and pagination.
// The search is performed on lesson title, description, and optionally on step content.
// It uses the withRetry method to handle transient errors.
//
// Parameters:
//   - opts: Search options including query, filters, pagination, and sorting
//
// Returns:
//   - A SearchResult containing the search results and metadata
//   - An error if the operation fails
func (s *MongoLessonStore) SearchLessons(opts SearchOptions) (*SearchResult, error) {
	var result SearchResult

	// Use default pagination if not specified
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 20 // Default page size
	}

	// Calculate skip value for pagination
	skip := (opts.Page - 1) * opts.PageSize

	err := s.withRetry("SearchLessons", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Build the search query
		filter := bson.M{}

		// Text search on title and description
		if opts.Query != "" {
			// If we're including content, we need to use $or to search across multiple fields
			if opts.IncludeContent {
				filter["$or"] = []bson.M{
					{"title": bson.M{"$regex": opts.Query, "$options": "i"}},
					{"description": bson.M{"$regex": opts.Query, "$options": "i"}},
					{"steps.content": bson.M{"$regex": opts.Query, "$options": "i"}},
				}
			} else {
				// Otherwise, just search title and description
				filter["$or"] = []bson.M{
					{"title": bson.M{"$regex": opts.Query, "$options": "i"}},
					{"description": bson.M{"$regex": opts.Query, "$options": "i"}},
				}
			}
		}

		// Filter by categories (OR logic)
		if len(opts.Categories) > 0 {
			filter["category"] = bson.M{"$in": opts.Categories}
		}

		// Filter by tags (OR logic)
		if len(opts.Tags) > 0 {
			filter["tags"] = bson.M{"$in": opts.Tags}
		}

		// Filter by required tags (AND logic)
		if len(opts.RequiredTags) > 0 {
			filter["tags"] = bson.M{"$all": opts.RequiredTags}
		}

		// Filter by difficulty
		if opts.Difficulty != "" {
			filter["difficulty"] = opts.Difficulty
		}

		// Filter by estimated time range
		if opts.MinEstimatedTime > 0 || opts.MaxEstimatedTime > 0 {
			timeFilter := bson.M{}
			if opts.MinEstimatedTime > 0 {
				timeFilter["$gte"] = opts.MinEstimatedTime
			}
			if opts.MaxEstimatedTime > 0 {
				timeFilter["$lte"] = opts.MaxEstimatedTime
			}
			filter["estimatedTime"] = timeFilter
		}

		// Create sort specification
		sortBson := bson.D{}
		if len(opts.Sort) > 0 {
			for k, v := range opts.Sort {
				sortBson = append(sortBson, bson.E{Key: k, Value: v})
			}
		} else {
			// Default sort by title ascending
			sortBson = append(sortBson, bson.E{Key: "title", Value: 1})
		}

		// Count total documents for pagination metadata
		totalItems, err := s.db.Collection("lessons").CountDocuments(ctx, filter)
		if err != nil {
			return err
		}

		// Configure find options
		findOptions := options.Find().
			SetSkip(skip).
			SetLimit(opts.PageSize).
			SetSort(sortBson)

		// Execute query
		cursor, err := s.db.Collection("lessons").Find(ctx, filter, findOptions)
		if err != nil {
			return err
		}
		defer cursor.Close(ctx)

		// Decode results
		var lessons []lesson.Lesson
		if err = cursor.All(ctx, &lessons); err != nil {
			return err
		}

		// Calculate pagination metadata
		totalPages := totalItems / opts.PageSize
		if totalItems%opts.PageSize > 0 {
			totalPages++
		}

		// Populate result
		result = SearchResult{
			Items:      lessons,
			TotalItems: totalItems,
			TotalPages: totalPages,
			Page:       opts.Page,
			PageSize:   opts.PageSize,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}
