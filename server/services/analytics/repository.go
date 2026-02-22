package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// Repository defines how analytics data is queried.
type Repository interface {
	GetCourseAnalytics(ctx context.Context, courseID string) (CourseStats, error)
	GetUserAnalytics(ctx context.Context, userID string) (UserStats, error)
	GetTenantDashboard(ctx context.Context, tenantID string) (TenantStats, error)
	IndexEvent(ctx context.Context, indexName string, doc map[string]interface{}) error
}

type CourseStats struct {
	TotalEnrollments int32
	ActiveStudents   int32
	Completions      int32
	AverageProgress  float64
}

type UserStats struct {
	CoursesEnrolled    int32
	CoursesCompleted   int32
	LearningStreakDays int32
	TotalLearningHours int32
}

type TenantStats struct {
	TotalUsers        int32
	TotalCourses      int32
	TotalEnrollments  int32
	ActiveUsersLast30 int32
}

// ElasticsearchRepository implements Repository using Elasticsearch.
type ElasticsearchRepository struct {
	es *elasticsearch.Client
}

func NewElasticsearchRepository(es *elasticsearch.Client) *ElasticsearchRepository {
	return &ElasticsearchRepository{es: es}
}

func (r *ElasticsearchRepository) IndexEvent(ctx context.Context, indexName string, doc map[string]interface{}) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := r.es.Index(indexName, bytes.NewReader(body),
		r.es.Index.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch index error: %s", res.Status())
	}
	return nil
}

func (r *ElasticsearchRepository) GetCourseAnalytics(ctx context.Context, courseID string) (CourseStats, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{"course_id.keyword": courseID},
		},
		"aggs": map[string]interface{}{
			"total_enrollments": map[string]interface{}{
				"value_count": map[string]interface{}{"field": "user_id.keyword"},
			},
			"completions": map[string]interface{}{
				"filter": map[string]interface{}{
					"term": map[string]interface{}{"event": "course.completed"},
				},
			},
			"avg_progress": map[string]interface{}{
				"avg": map[string]interface{}{"field": "progress"},
			},
		},
		"size": 0,
	}

	raw, err := r.runQuery(ctx, "analytics-enrollments", query)
	if err != nil {
		return CourseStats{}, err
	}

	aggs := raw["aggregations"].(map[string]interface{})
	stats := CourseStats{
		TotalEnrollments: int32(aggs["total_enrollments"].(map[string]interface{})["value"].(float64)),
		Completions:      int32(aggs["completions"].(map[string]interface{})["doc_count"].(float64)),
	}
	if avg, ok := aggs["avg_progress"].(map[string]interface{})["value"].(float64); ok {
		stats.AverageProgress = avg
	}
	// ActiveStudents ≈ enrolled who haven't completed
	stats.ActiveStudents = stats.TotalEnrollments - stats.Completions
	return stats, nil
}

func (r *ElasticsearchRepository) GetUserAnalytics(ctx context.Context, userID string) (UserStats, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{"user_id.keyword": userID},
		},
		"aggs": map[string]interface{}{
			"enrolled": map[string]interface{}{
				"filter": map[string]interface{}{
					"term": map[string]interface{}{"event": "enrollment.created"},
				},
			},
			"completed": map[string]interface{}{
				"filter": map[string]interface{}{
					"term": map[string]interface{}{"event": "course.completed"},
				},
			},
		},
		"size": 0,
	}

	raw, err := r.runQuery(ctx, "analytics-enrollments", query)
	if err != nil {
		return UserStats{}, err
	}

	aggs := raw["aggregations"].(map[string]interface{})
	return UserStats{
		CoursesEnrolled:  int32(aggs["enrolled"].(map[string]interface{})["doc_count"].(float64)),
		CoursesCompleted: int32(aggs["completed"].(map[string]interface{})["doc_count"].(float64)),
		// Streak and hours require richer event data; default to 0 until tracked.
		LearningStreakDays: 0,
		TotalLearningHours: 0,
	}, nil
}

func (r *ElasticsearchRepository) GetTenantDashboard(ctx context.Context, tenantID string) (TenantStats, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{"tenant_id.keyword": tenantID},
		},
		"aggs": map[string]interface{}{
			"total_users": map[string]interface{}{
				"cardinality": map[string]interface{}{"field": "user_id.keyword"},
			},
			"total_courses": map[string]interface{}{
				"cardinality": map[string]interface{}{"field": "course_id.keyword"},
			},
			"total_enrollments": map[string]interface{}{
				"filter": map[string]interface{}{
					"term": map[string]interface{}{"event": "enrollment.created"},
				},
			},
		},
		"size": 0,
	}

	raw, err := r.runQuery(ctx, "analytics-enrollments", query)
	if err != nil {
		return TenantStats{}, err
	}

	aggs := raw["aggregations"].(map[string]interface{})
	return TenantStats{
		TotalUsers:        int32(aggs["total_users"].(map[string]interface{})["value"].(float64)),
		TotalCourses:      int32(aggs["total_courses"].(map[string]interface{})["value"].(float64)),
		TotalEnrollments:  int32(aggs["total_enrollments"].(map[string]interface{})["doc_count"].(float64)),
		ActiveUsersLast30: 0, // requires date_histogram aggregation — added in v2
	}, nil
}

// runQuery executes an Elasticsearch DSL query and returns the raw response map.
func (r *ElasticsearchRepository) runQuery(ctx context.Context, index string, query map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	res, err := r.es.Search(
		r.es.Search.WithContext(ctx),
		r.es.Search.WithIndex(index),
		r.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch search error: %s", res.Status())
	}

	rawBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.NewDecoder(strings.NewReader(string(rawBytes))).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}
