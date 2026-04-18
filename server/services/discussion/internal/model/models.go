package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Thread represents a discussion thread
type Thread struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID       string             `bson:"tenant_id" json:"tenant_id"`
	CourseID       string             `bson:"course_id" json:"course_id"`
	ModuleID       string             `bson:"module_id" json:"module_id"`
	LessonID       string             `bson:"lesson_id" json:"lesson_id"`
	AuthorID       string             `bson:"author_id" json:"author_id"`
	AuthorName     string             `bson:"author_name" json:"author_name"`
	AuthorAvatar   string             `bson:"author_avatar" json:"author_avatar"`
	Title          string             `bson:"title" json:"title"`
	Content        string             `bson:"content" json:"content"`
	Tags           []string           `bson:"tags" json:"tags"`
	Upvotes        int32              `bson:"upvotes" json:"upvotes"`
	Downvotes      int32              `bson:"downvotes" json:"downvotes"`
	ReplyCount     int32              `bson:"reply_count" json:"reply_count"`
	IsPinned       bool               `bson:"is_pinned" json:"is_pinned"`
	IsResolved     bool               `bson:"is_resolved" json:"is_resolved"`
	ResolvedReplyID string            `bson:"resolved_reply_id,omitempty" json:"resolved_reply_id"`
	IsSubscribed   bool               `bson:"-" json:"-"` // Computed at query time
	HasVoted       bool               `bson:"-" json:"has_voted"`
	UserVoteType   int32              `bson:"-" json:"user_vote_type"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
	DeletedAt      *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at"`
	LastActivityAt time.Time          `bson:"last_activity_at" json:"last_activity_at"`
	Category       string             `bson:"category" json:"category"` // general, question, feedback, announcement
}

// Reply represents a reply to a thread
type Reply struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ThreadID      string              `bson:"thread_id" json:"thread_id"`
	TenantID      string              `bson:"tenant_id" json:"tenant_id"`
	ParentReplyID *primitive.ObjectID `bson:"parent_reply_id,omitempty" json:"parent_reply_id"`
	AuthorID      string              `bson:"author_id" json:"author_id"`
	AuthorName    string              `bson:"author_name" json:"author_name"`
	AuthorAvatar  string              `bson:"author_avatar" json:"author_avatar"`
	Content       string              `bson:"content" json:"content"`
	Upvotes       int32               `bson:"upvotes" json:"upvotes"`
	Downvotes     int32               `bson:"downvotes" json:"downvotes"`
	IsAccepted    bool                `bson:"is_accepted" json:"is_accepted"`
	HasVoted      bool                `bson:"-" json:"has_voted"`
	UserVoteType  int32               `bson:"-" json:"user_vote_type"`
	CreatedAt     time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time           `bson:"updated_at" json:"updated_at"`
	DeletedAt     *time.Time          `bson:"deleted_at,omitempty" json:"deleted_at"`
	Replies       []Reply             `bson:"replies,omitempty" json:"replies"`
}

// Vote represents a vote on a thread or reply
type Vote struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID  string             `bson:"tenant_id" json:"tenant_id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	ResourceType string          `bson:"resource_type" json:"resource_type"` // thread, reply
	ResourceID  string           `bson:"resource_id" json:"resource_id"`
	VoteType    int32            `bson:"vote_type" json:"vote_type"` // 1: up, -1: down
	CreatedAt   time.Time        `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `bson:"updated_at" json:"updated_at"`
}

// Subscription represents a user's subscription to a thread
type Subscription struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID  string             `bson:"tenant_id" json:"tenant_id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	ThreadID  string             `bson:"thread_id" json:"thread_id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// UserStats represents a user's discussion statistics
type UserStats struct {
	UserID              string  `bson:"user_id" json:"user_id"`
	TenantID            string  `bson:"tenant_id" json:"tenant_id"`
	ThreadsCreated      int32   `bson:"threads_created" json:"threads_created"`
	RepliesCreated      int32   `bson:"replies_created" json:"replies_created"`
	TotalUpvotes        int32   `bson:"total_upvotes" json:"total_upvotes"`
	TotalDownvotes      int32   `bson:"total_downvotes" json:"total_downvotes"`
	SolutionsProvided   int32   `bson:"solutions_provided" json:"solutions_provided"`
	BestAnswerCount     int32   `bson:"best_answer_count" json:"best_answer_count"`
	ReputationScore     float64 `bson:"reputation_score" json:"reputation_score"`
	LastCalculatedAt    time.Time `bson:"last_calculated_at" json:"last_calculated_at"`
}

// TableName returns the collection name for threads
func (Thread) CollectionName() string {
	return "discussion_threads"
}

// TableName returns the collection name for replies
func (Reply) CollectionName() string {
	return "discussion_replies"
}

// TableName returns the collection name for votes
func (Vote) CollectionName() string {
	return "discussion_votes"
}

// TableName returns the collection name for subscriptions
func (Subscription) CollectionName() string {
	return "discussion_subscriptions"
}

// TableName returns the collection name for user stats
func (UserStats) CollectionName() string {
	return "discussion_user_stats"
}
