package store

import (
	"context"
	"lessoncraft/lesson"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoLessonStore struct {
	db *mongo.Database
}

func NewMongoLessonStore(db *mongo.Database) *MongoLessonStore {
	return &MongoLessonStore{db: db}
}

func (s *MongoLessonStore) ListLessons() ([]lesson.Lesson, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.db.Collection("lessons").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var lessons []lesson.Lesson
	if err = cursor.All(ctx, &lessons); err != nil {
		return nil, err
	}
	return lessons, nil
}

func (s *MongoLessonStore) GetLesson(id string) (*lesson.Lesson, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var lesson lesson.Lesson
	err := s.db.Collection("lessons").FindOne(ctx, bson.M{"id": id}).Decode(&lesson)
	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

func (s *MongoLessonStore) CreateLesson(l *lesson.Lesson) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	l.ID = uuid.New().String()
	l.CreatedAt = time.Now()

	_, err := s.db.Collection("lessons").InsertOne(ctx, l)
	return err
}

func (s *MongoLessonStore) UpdateLesson(id string, l *lesson.Lesson) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.db.Collection("lessons").UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.M{"$set": l},
	)
	return err
}

func (s *MongoLessonStore) DeleteLesson(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.db.Collection("lessons").DeleteOne(ctx, bson.M{"id": id})
	return err
}
