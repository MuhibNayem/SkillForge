package content

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ContentRecord struct {
	ID          string    `bson:"_id,omitempty"`
	Filename    string    `bson:"filename"`
	ContentType string    `bson:"content_type"`
	SizeBytes   int64     `bson:"size_bytes"`
	TenantID    string    `bson:"tenant_id"`
	CourseID    string    `bson:"course_id"`
	UploadedBy  string    `bson:"uploaded_by"`
	Status      string    `bson:"status"`
	CreatedAt   time.Time `bson:"created_at"`
}

type Repository interface {
	Store(ctx context.Context, rec ContentRecord) (ContentRecord, error)
	GetByID(ctx context.Context, id string) (ContentRecord, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, tenantID, courseID string, page, pageSize int) ([]ContentRecord, int, error)
}

var ErrContentNotFound = errors.New("content not found")

type MongoRepository struct {
	col      *mongo.Collection
	minioURL string
}

func NewMongoRepository(db *mongo.Database, minioURL string) *MongoRepository {
	return &MongoRepository{
		col:      db.Collection("content_metadata"),
		minioURL: minioURL,
	}
}

func (r *MongoRepository) Store(ctx context.Context, rec ContentRecord) (ContentRecord, error) {
	rec.CreatedAt = time.Now()
	rec.Status = "uploaded"
	result, err := r.col.InsertOne(ctx, rec)
	if err != nil {
		return rec, err
	}
	// InsertedID is bson.ObjectID — extract the hex string
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		rec.ID = oid.Hex()
	}
	return rec, nil
}

func (r *MongoRepository) GetByID(ctx context.Context, id string) (ContentRecord, error) {
	var rec ContentRecord
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return rec, ErrContentNotFound
	}
	err = r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&rec)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return rec, ErrContentNotFound
	}
	return rec, err
}

func (r *MongoRepository) Delete(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrContentNotFound
	}
	result, err := r.col.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrContentNotFound
	}
	return nil
}

func (r *MongoRepository) List(ctx context.Context, tenantID, courseID string, page, pageSize int) ([]ContentRecord, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	filter := bson.M{"tenant_id": tenantID}
	if courseID != "" {
		filter["course_id"] = courseID
	}

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.M{"created_at": -1})
	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var records []ContentRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, 0, err
	}
	return records, int(total), nil
}
