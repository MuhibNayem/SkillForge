package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/learnhub/lms/server/services/discussion/internal/model"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrThreadNotFound  = errors.New("thread not found")
	ErrReplyNotFound   = errors.New("reply not found")
	ErrVoteExists      = errors.New("vote already exists")
	ErrSubscriptionExists = errors.New("subscription already exists")
)

// Repository defines the interface for discussion data operations
type Repository interface {
	// Thread operations
	CreateThread(ctx context.Context, thread *model.Thread) (*model.Thread, error)
	GetThread(ctx context.Context, id string) (*model.Thread, error)
	ListThreads(ctx context.Context, filter ThreadFilter) ([]*model.Thread, int64, error)
	UpdateThread(ctx context.Context, thread *model.Thread) (*model.Thread, error)
	DeleteThread(ctx context.Context, id string) error
	PinThread(ctx context.Context, id string, pinned bool) (*model.Thread, error)
	ResolveThread(ctx context.Context, threadID, replyID string) (*model.Thread, error)
	
	// Reply operations
	CreateReply(ctx context.Context, reply *model.Reply) (*model.Reply, error)
	ListReplies(ctx context.Context, threadID string, filter ReplyFilter) ([]*model.Reply, int64, error)
	UpdateReply(ctx context.Context, reply *model.Reply) (*model.Reply, error)
	DeleteReply(ctx context.Context, id string) error
	
	// Vote operations
	Vote(ctx context.Context, vote *model.Vote) (int32, int32, int32, error)
	GetUserVote(ctx context.Context, userID, resourceType, resourceID string) (int32, error)
	
	// Subscription operations
	Subscribe(ctx context.Context, sub *model.Subscription) error
	Unsubscribe(ctx context.Context, userID, threadID string) error
	IsSubscribed(ctx context.Context, userID, threadID string) (bool, error)
	
	// User stats
	GetUserStats(ctx context.Context, userID, tenantID string) (*model.UserStats, error)
	UpdateUserStats(ctx context.Context, stats *model.UserStats) error
	
	// Cache operations
	CacheThread(ctx context.Context, thread *model.Thread) error
	GetCachedThread(ctx context.Context, id string) (*model.Thread, error)
	InvalidateThreadCache(ctx context.Context, id string) error
}

// ThreadFilter contains filtering options for listing threads
type ThreadFilter struct {
	TenantID         string
	CourseID         string
	ModuleID         string
	LessonID         string
	Category         string
	Search           string
	Tags             []string
	IncludePinnedFirst bool
	ShowResolved     bool
	SortBy           string
	SortOrder        string
	Page             int32
	PageSize         int32
}

// ReplyFilter contains filtering options for listing replies
type ReplyFilter struct {
	SortBy    string
	SortOrder string
	Page      int32
	PageSize  int32
}

// MongoRepository implements Repository using MongoDB and Redis
type MongoRepository struct {
	mongoClient *mongo.Client
	redisClient *redis.Client
	threadsColl *mongo.Collection
	repliesColl *mongo.Collection
	votesColl   *mongo.Collection
	subsColl    *mongo.Collection
	statsColl   *mongo.Collection
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository(mongoClient *mongo.Client, redisClient *redis.Client) Repository {
	db := mongoClient.Database("learnhub_discussion")
	return &MongoRepository{
		mongoClient: mongoClient,
		redisClient: redisClient,
		threadsColl: db.Collection(model.Thread{}.CollectionName()),
		repliesColl: db.Collection(model.Reply{}.CollectionName()),
		votesColl:   db.Collection(model.Vote{}.CollectionName()),
		subsColl:    db.Collection(model.Subscription{}.CollectionName()),
		statsColl:   db.Collection(model.UserStats{}.CollectionName()),
	}
}

// CreateThread creates a new thread
func (r *MongoRepository) CreateThread(ctx context.Context, thread *model.Thread) (*model.Thread, error) {
	thread.ID = primitive.NewObjectID()
	thread.CreatedAt = time.Now()
	thread.UpdatedAt = time.Now()
	thread.LastActivityAt = time.Now()
	
	if _, err := r.threadsColl.InsertOne(ctx, thread); err != nil {
		return nil, fmt.Errorf("failed to insert thread: %w", err)
	}
	
	// Initialize user stats
	if err := r.incrementUserStat(ctx, thread.TenantID, thread.AuthorID, "threads_created"); err != nil {
		// Log error but don't fail the operation
	}
	
	return thread, nil
}

// GetThread retrieves a thread by ID
func (r *MongoRepository) GetThread(ctx context.Context, id string) (*model.Thread, error) {
	// Try cache first
	if cached, err := r.GetCachedThread(ctx, id); err == nil {
		return cached, nil
	}
	
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrThreadNotFound
	}
	
	var thread model.Thread
	err = r.threadsColl.FindOne(ctx, bson.M{
		"_id": objectID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&thread)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrThreadNotFound
		}
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}
	
	// Cache the thread
	r.CacheThread(ctx, &thread)
	
	return &thread, nil
}

// ListThreads lists threads with filtering and pagination
func (r *MongoRepository) ListThreads(ctx context.Context, filter ThreadFilter) ([]*model.Thread, int64, error) {
	query := bson.M{"deleted_at": bson.M{"$exists": false}}
	
	if filter.TenantID != "" {
		query["tenant_id"] = filter.TenantID
	}
	if filter.CourseID != "" {
		query["course_id"] = filter.CourseID
	}
	if filter.ModuleID != "" {
		query["module_id"] = filter.ModuleID
	}
	if filter.LessonID != "" {
		query["lesson_id"] = filter.LessonID
	}
	if filter.Category != "" {
		query["category"] = filter.Category
	}
	if !filter.ShowResolved {
		query["is_resolved"] = false
	}
	if len(filter.Tags) > 0 {
		query["tags"] = bson.M{"$in": filter.Tags}
	}
	if filter.Search != "" {
		query["$or"] = []bson.M{
			{"title": bson.M{"$regex": filter.Search, "$options": "i"}},
			{"content": bson.M{"$regex": filter.Search, "$options": "i"}},
		}
	}
	
	// Set up sorting
	sortField := "created_at"
	sortOrder := -1 // desc
	switch filter.SortBy {
	case "updated_at":
		sortField = "updated_at"
	case "upvotes":
		sortField = "upvotes"
	case "reply_count":
		sortField = "reply_count"
	}
	if filter.SortOrder == "asc" {
		sortOrder = 1
	}
	
	// Pinned threads first if requested
	sortStage := bson.D{{Key: sortField, Value: sortOrder}}
	if filter.IncludePinnedFirst {
		sortStage = append(bson.D{{Key: "is_pinned", Value: -1}}, sortStage...)
	}
	
	// Pagination
	skip := int64((filter.Page - 1) * filter.PageSize)
	limit := int64(filter.PageSize)
	
	opts := options.Find().SetSort(sortStage).SetSkip(skip).SetLimit(limit)
	
	cursor, err := r.threadsColl.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list threads: %w", err)
	}
	defer cursor.Close(ctx)
	
	var threads []*model.Thread
	if err := cursor.All(ctx, &threads); err != nil {
		return nil, 0, fmt.Errorf("failed to decode threads: %w", err)
	}
	
	// Get total count
	totalCount, err := r.threadsColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count threads: %w", err)
	}
	
	return threads, totalCount, nil
}

// UpdateThread updates an existing thread
func (r *MongoRepository) UpdateThread(ctx context.Context, thread *model.Thread) (*model.Thread, error) {
	objectID, err := primitive.ObjectIDFromHex(thread.ID.Hex())
	if err != nil {
		return nil, ErrThreadNotFound
	}
	
	thread.UpdatedAt = time.Now()
	
	update := bson.M{
		"$set": bson.M{
			"title":        thread.Title,
			"content":      thread.Content,
			"tags":         thread.Tags,
			"category":     thread.Category,
			"updated_at":   thread.UpdatedAt,
		},
	}
	
	_, err = r.threadsColl.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update thread: %w", err)
	}
	
	// Invalidate cache
	r.InvalidateThreadCache(ctx, thread.ID.Hex())
	
	// Fetch updated thread
	return r.GetThread(ctx, thread.ID.Hex())
}

// DeleteThread soft deletes a thread
func (r *MongoRepository) DeleteThread(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrThreadNotFound
	}
	
	update := bson.M{"$set": bson.M{"deleted_at": time.Now()}}
	_, err = r.threadsColl.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}
	
	r.InvalidateThreadCache(ctx, id)
	return nil
}

// PinThread pins or unpins a thread
func (r *MongoRepository) PinThread(ctx context.Context, id string, pinned bool) (*model.Thread, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrThreadNotFound
	}
	
	update := bson.M{
		"$set": bson.M{
			"is_pinned":  pinned,
			"updated_at": time.Now(),
		},
	}
	
	_, err = r.threadsColl.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to pin thread: %w", err)
	}
	
	r.InvalidateThreadCache(ctx, id)
	return r.GetThread(ctx, id)
}

// ResolveThread marks a thread as resolved with optional solution reply
func (r *MongoRepository) ResolveThread(ctx context.Context, threadID, replyID string) (*model.Thread, error) {
	objectID, err := primitive.ObjectIDFromHex(threadID)
	if err != nil {
		return nil, ErrThreadNotFound
	}
	
	update := bson.M{
		"$set": bson.M{
			"is_resolved":       true,
			"resolved_reply_id": replyID,
			"updated_at":        time.Now(),
		},
	}
	
	_, err = r.threadsColl.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve thread: %w", err)
	}
	
	r.InvalidateThreadCache(ctx, threadID)
	return r.GetThread(ctx, threadID)
}

// CreateReply creates a new reply
func (r *MongoRepository) CreateReply(ctx context.Context, reply *model.Reply) (*model.Reply, error) {
	reply.ID = primitive.NewObjectID()
	reply.CreatedAt = time.Now()
	reply.UpdatedAt = time.Now()
	
	if _, err := r.repliesColl.InsertOne(ctx, reply); err != nil {
		return nil, fmt.Errorf("failed to insert reply: %w", err)
	}
	
	// Increment thread reply count
	threadOID, _ := primitive.ObjectIDFromHex(reply.ThreadID)
	_, err := r.threadsColl.UpdateOne(ctx, bson.M{"_id": threadOID}, bson.M{
		"$inc": bson.M{
			"reply_count":      1,
			"last_activity_at": time.Now(),
		},
		"$set": bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		// Log error but don't fail
	}
	
	// Initialize user stats
	if err := r.incrementUserStat(ctx, reply.TenantID, reply.AuthorID, "replies_created"); err != nil {
		// Log error but don't fail
	}
	
	// Invalidate thread cache
	r.InvalidateThreadCache(ctx, reply.ThreadID)
	
	return reply, nil
}

// ListReplies lists replies for a thread
func (r *MongoRepository) ListReplies(ctx context.Context, threadID string, filter ReplyFilter) ([]*model.Reply, int64, error) {
	query := bson.M{
		"thread_id":  threadID,
		"deleted_at": bson.M{"$exists": false},
	}
	
	// Only top-level replies if no parent filter
	query["parent_reply_id"] = bson.M{"$exists": false}
	
	// Sorting
	sortField := "created_at"
	sortOrder := 1 // asc (oldest first)
	switch filter.SortBy {
	case "upvotes":
		sortField = "upvotes"
		sortOrder = -1
	}
	if filter.SortOrder == "desc" && filter.SortBy == "created_at" {
		sortOrder = -1
	}
	
	// Pagination
	skip := int64((filter.Page - 1) * filter.PageSize)
	limit := int64(filter.PageSize)
	
	opts := options.Find().SetSort(bson.D{{Key: sortField, Value: sortOrder}}).SetSkip(skip).SetLimit(limit)
	
	cursor, err := r.repliesColl.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list replies: %w", err)
	}
	defer cursor.Close(ctx)
	
	var replies []*model.Reply
	if err := cursor.All(ctx, &replies); err != nil {
		return nil, 0, fmt.Errorf("failed to decode replies: %w", err)
	}
	
	// Get nested replies for each top-level reply
	for _, reply := range replies {
		nestedQuery := bson.M{
			"parent_reply_id": reply.ID,
			"deleted_at":      bson.M{"$exists": false},
		}
		nestedOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
		nestedCursor, err := r.repliesColl.Find(ctx, nestedQuery, nestedOpts)
		if err == nil {
			var nested []*model.Reply
			if nestedCursor.All(ctx, &nested) == nil {
				reply.Replies = nested
			}
			nestedCursor.Close(ctx)
		}
	}
	
	totalCount, err := r.repliesColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count replies: %w", err)
	}
	
	return replies, totalCount, nil
}

// UpdateReply updates an existing reply
func (r *MongoRepository) UpdateReply(ctx context.Context, reply *model.Reply) (*model.Reply, error) {
	objectID, err := primitive.ObjectIDFromHex(reply.ID.Hex())
	if err != nil {
		return nil, ErrReplyNotFound
	}
	
	reply.UpdatedAt = time.Now()
	
	update := bson.M{
		"$set": bson.M{
			"content":    reply.Content,
			"is_edited":  true,
			"updated_at": reply.UpdatedAt,
		},
	}
	
	_, err = r.repliesColl.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update reply: %w", err)
	}
	
	// Invalidate thread cache
	r.InvalidateThreadCache(ctx, reply.ThreadID)
	
	return r.GetReply(ctx, reply.ID.Hex())
}

// GetReply retrieves a single reply
func (r *MongoRepository) GetReply(ctx context.Context, id string) (*model.Reply, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrReplyNotFound
	}
	
	var reply model.Reply
	err = r.repliesColl.FindOne(ctx, bson.M{
		"_id": objectID,
		"deleted_at": bson.M{"$exists": false},
	}).Decode(&reply)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrReplyNotFound
		}
		return nil, fmt.Errorf("failed to get reply: %w", err)
	}
	
	return &reply, nil
}

// DeleteReply soft deletes a reply
func (r *MongoRepository) DeleteReply(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrReplyNotFound
	}
	
	// Get reply to find thread ID
	var reply model.Reply
	err = r.repliesColl.FindOne(ctx, bson.M{"_id": objectID}).Decode(&reply)
	if err != nil {
		return ErrReplyNotFound
	}
	
	update := bson.M{"$set": bson.M{"deleted_at": time.Now()}}
	_, err = r.repliesColl.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return fmt.Errorf("failed to delete reply: %w", err)
	}
	
	// Decrement thread reply count
	threadOID, _ := primitive.ObjectIDFromHex(reply.ThreadID)
	r.threadsColl.UpdateOne(ctx, bson.M{"_id": threadOID}, bson.M{
		"$inc": bson.M{"reply_count": -1},
	})
	
	r.InvalidateThreadCache(ctx, reply.ThreadID)
	return nil
}

// Vote handles voting on threads and replies
func (r *MongoRepository) Vote(ctx context.Context, vote *model.Vote) (int32, int32, int32, error) {
	vote.ID = primitive.NewObjectID()
	now := time.Now()
	vote.CreatedAt = now
	vote.UpdatedAt = now
	
	// Check if user already voted
	existingVote, err := r.GetUserVote(ctx, vote.UserID, vote.ResourceType, vote.ResourceID)
	if err != nil && !errors.Is(err, ErrVoteExists) {
		return 0, 0, 0, err
	}
	
	coll := r.votesColl
	resourceColl := r.threadsColl
	if vote.ResourceType == "reply" {
		resourceColl = r.repliesColl
	}
	
	objectID, _ := primitive.ObjectIDFromHex(vote.ResourceID)
	
	session, err := r.mongoClient.StartSession()
	if err != nil {
		return 0, 0, 0, err
	}
	defer session.EndSession(ctx)
	
	var upvotes, downvotes, userVote int32
	
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if existingVote != 0 {
			// Update existing vote
			if vote.VoteType == 0 {
				// Remove vote
				coll.DeleteOne(sessCtx, bson.M{
					"user_id": vote.UserID,
					"resource_type": vote.ResourceType,
					"resource_id": vote.ResourceID,
				})
				
				// Adjust counts
				delta := -existingVote
				if vote.ResourceType == "thread" {
					if existingVote == 1 {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"upvotes": -1}})
					} else {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"downvotes": -1}})
					}
				} else {
					if existingVote == 1 {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"upvotes": -1}})
					} else {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"downvotes": -1}})
					}
				}
			} else {
				// Change vote
				coll.UpdateOne(sessCtx, bson.M{
					"user_id": vote.UserID,
					"resource_type": vote.ResourceType,
					"resource_id": vote.ResourceID,
				}, bson.M{
					"$set": bson.M{"vote_type": vote.VoteType, "updated_at": now},
				})
				
				// Adjust counts
				if vote.ResourceType == "thread" {
					if existingVote == 1 && vote.VoteType == -1 {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"upvotes": -1, "downvotes": 1}})
					} else if existingVote == -1 && vote.VoteType == 1 {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"upvotes": 1, "downvotes": -1}})
					}
				}
			}
		} else {
			// New vote
			if vote.VoteType != 0 {
				coll.InsertOne(sessCtx, vote)
				
				if vote.ResourceType == "thread" {
					if vote.VoteType == 1 {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"upvotes": 1}})
					} else {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"downvotes": 1}})
					}
				} else {
					if vote.VoteType == 1 {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"upvotes": 1}})
					} else {
						coll.UpdateOne(sessCtx, bson.M{"_id": objectID}, bson.M{"$inc": bson.M{"downvotes": 1}})
					}
				}
			}
		}
		
		// Get updated counts
		var resourceDoc bson.M
		var targetColl *mongo.Collection
		if vote.ResourceType == "thread" {
			targetColl = r.threadsColl
		} else {
			targetColl = r.repliesColl
		}
		
		err := targetColl.FindOne(sessCtx, bson.M{"_id": objectID}).Decode(&resourceDoc)
		if err != nil {
			return nil, err
		}
		
		upvotes = int32(resourceDoc["upvotes"].(int32))
		downvotes = int32(resourceDoc["downvotes"].(int32))
		userVote = vote.VoteType
		
		return nil, nil
	})
	
	if err != nil {
		return 0, 0, 0, err
	}
	
	// Invalidate cache
	r.InvalidateThreadCache(ctx, vote.ResourceID)
	
	return upvotes, downvotes, userVote, nil
}

// GetUserVote gets a user's vote on a resource
func (r *MongoRepository) GetUserVote(ctx context.Context, userID, resourceType, resourceID string) (int32, error) {
	var vote model.Vote
	err := r.votesColl.FindOne(ctx, bson.M{
		"user_id": userID,
		"resource_type": resourceType,
		"resource_id": resourceID,
	}).Decode(&vote)
	
	if err == mongo.ErrNoDocuments {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	
	return vote.VoteType, nil
}

// Subscribe subscribes a user to a thread
func (r *MongoRepository) Subscribe(ctx context.Context, sub *model.Subscription) error {
	sub.ID = primitive.NewObjectID()
	sub.CreatedAt = time.Now()
	
	_, err := r.subsColl.InsertOne(ctx, sub)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	
	return nil
}

// Unsubscribe unsubscribes a user from a thread
func (r *MongoRepository) Unsubscribe(ctx context.Context, userID, threadID string) error {
	_, err := r.subsColl.DeleteMany(ctx, bson.M{
		"user_id": userID,
		"thread_id": threadID,
	})
	return err
}

// IsSubscribed checks if a user is subscribed to a thread
func (r *MongoRepository) IsSubscribed(ctx context.Context, userID, threadID string) (bool, error) {
	count, err := r.subsColl.CountDocuments(ctx, bson.M{
		"user_id": userID,
		"thread_id": threadID,
	})
	return count > 0, err
}

// GetUserStats gets statistics for a user
func (r *MongoRepository) GetUserStats(ctx context.Context, userID, tenantID string) (*model.UserStats, error) {
	var stats model.UserStats
	err := r.statsColl.FindOne(ctx, bson.M{
		"user_id": userID,
		"tenant_id": tenantID,
	}).Decode(&stats)
	
	if err == mongo.ErrNoDocuments {
		// Calculate stats on the fly
		return r.calculateUserStats(ctx, userID, tenantID)
	}
	if err != nil {
		return nil, err
	}
	
	return &stats, nil
}

// UpdateUserStats updates user statistics
func (r *MongoRepository) UpdateUserStats(ctx context.Context, stats *model.UserStats) error {
	stats.LastCalculatedAt = time.Now()
	
	_, err := r.statsColl.UpdateOne(ctx, bson.M{
		"user_id": stats.UserID,
		"tenant_id": stats.TenantID,
	}, bson.M{"$set": stats}, options.Update().SetUpsert(true))
	
	return err
}

// CacheThread caches a thread in Redis
func (r *MongoRepository) CacheThread(ctx context.Context, thread *model.Thread) error {
	key := fmt.Sprintf("thread:%s", thread.ID.Hex())
	data, err := bson.Marshal(thread)
	if err != nil {
		return err
	}
	
	return r.redisClient.Set(ctx, key, data, 5*time.Minute).Err()
}

// GetCachedThread retrieves a cached thread from Redis
func (r *MongoRepository) GetCachedThread(ctx context.Context, id string) (*model.Thread, error) {
	key := fmt.Sprintf("thread:%s", id)
	data, err := r.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	
	var thread model.Thread
	err = bson.Unmarshal(data, &thread)
	if err != nil {
		return nil, err
	}
	
	return &thread, nil
}

// InvalidateThreadCache invalidates a thread cache
func (r *MongoRepository) InvalidateThreadCache(ctx context.Context, id string) error {
	key := fmt.Sprintf("thread:%s", id)
	return r.redisClient.Del(ctx, key).Err()
}

// Helper methods

func (r *MongoRepository) incrementUserStat(ctx context.Context, tenantID, userID, field string) error {
	_, err := r.statsColl.UpdateOne(ctx, bson.M{
		"user_id": userID,
		"tenant_id": tenantID,
	}, bson.M{
		"$inc": bson.M{field: 1},
		"$set": bson.M{"last_calculated_at": time.Now()},
	}, options.Update().SetUpsert(true))
	return err
}

func (r *MongoRepository) calculateUserStats(ctx context.Context, userID, tenantID string) (*model.UserStats, error) {
	stats := &model.UserStats{
		UserID:   userID,
		TenantID: tenantID,
	}
	
	// Count threads
	threadCount, _ := r.threadsColl.CountDocuments(ctx, bson.M{
		"author_id": userID,
		"tenant_id": tenantID,
		"deleted_at": bson.M{"$exists": false},
	})
	stats.ThreadsCreated = int32(threadCount)
	
	// Count replies
	replyCount, _ := r.repliesColl.CountDocuments(ctx, bson.M{
		"author_id": userID,
		"tenant_id": tenantID,
		"deleted_at": bson.M{"$exists": false},
	})
	stats.RepliesCreated = int32(replyCount)
	
	// Calculate upvotes received
	cursor, _ := r.threadsColl.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"author_id": userID, "tenant_id": tenantID}}},
		bson.D{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$upvotes"}}}},
	})
	defer cursor.Close(ctx)
	if cursor.Next(ctx) {
		var result struct{ Total int32 }
		cursor.Decode(&result)
		stats.TotalUpvotes = result.Total
	}
	
	// Reputation score calculation
	stats.ReputationScore = float64(stats.TotalUpvotes)*10 + 
		float64(stats.ThreadsCreated)*5 + 
		float64(stats.RepliesCreated)*2
	
	stats.LastCalculatedAt = time.Now()
	
	// Save calculated stats
	r.UpdateUserStats(ctx, stats)
	
	return stats, nil
}
