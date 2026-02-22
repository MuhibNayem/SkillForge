package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// SearchResult mirrors what Elasticsearch returns per-hit.
type SearchResult struct {
	ID             string
	Type           string
	Title          string
	Description    string
	URL            string
	RelevanceScore float64
}

// Repository defines how the search index is queried and populated.
type Repository interface {
	Search(ctx context.Context, query, tenantID string, filters []string, limit, offset int) ([]SearchResult, int, error)
	IndexDocument(ctx context.Context, doc IndexDocument) error
}

// IndexDocument is the shape of a document written to the search index.
type IndexDocument struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // "course" | "user" | "lesson"
	Title       string `json:"title"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id"`
	URL         string `json:"url"`
}

// ElasticsearchRepository implements Repository.
type ElasticsearchRepository struct {
	es    *elasticsearch.Client
	index string
}

func NewElasticsearchRepository(es *elasticsearch.Client) *ElasticsearchRepository {
	return &ElasticsearchRepository{es: es, index: "skillforge-search"}
}

func (r *ElasticsearchRepository) IndexDocument(ctx context.Context, doc IndexDocument) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := r.es.Index(r.index, bytes.NewReader(body),
		r.es.Index.WithContext(ctx),
		r.es.Index.WithDocumentID(doc.ID),
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

func (r *ElasticsearchRepository) Search(ctx context.Context, query, tenantID string, filters []string, limit, offset int) ([]SearchResult, int, error) {
	if limit <= 0 {
		limit = 10
	}

	mustClauses := []interface{}{
		map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     query,
				"fields":    []string{"title^3", "description"},
				"fuzziness": "AUTO",
			},
		},
		map[string]interface{}{
			"term": map[string]interface{}{"tenant_id.keyword": tenantID},
		},
	}

	// Append type filters (e.g. "type:course").
	for _, f := range filters {
		if strings.HasPrefix(f, "type:") {
			mustClauses = append(mustClauses, map[string]interface{}{
				"term": map[string]interface{}{"type.keyword": strings.TrimPrefix(f, "type:")},
			})
		}
	}

	dslQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{"must": mustClauses},
		},
		"from": offset,
		"size": limit,
	}

	body, err := json.Marshal(dslQuery)
	if err != nil {
		return nil, 0, err
	}
	res, err := r.es.Search(
		r.es.Search.WithContext(ctx),
		r.es.Search.WithIndex(r.index),
		r.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("elasticsearch search error: %s", res.Status())
	}

	rawBytes, _ := io.ReadAll(res.Body)
	var raw map[string]interface{}
	if err := json.NewDecoder(strings.NewReader(string(rawBytes))).Decode(&raw); err != nil {
		return nil, 0, err
	}

	hitsWrapper, _ := raw["hits"].(map[string]interface{})
	totalObj, _ := hitsWrapper["total"].(map[string]interface{})
	totalHits := int(totalObj["value"].(float64))

	hitsArr, _ := hitsWrapper["hits"].([]interface{})
	var results []SearchResult
	for _, h := range hitsArr {
		hit := h.(map[string]interface{})
		src := hit["_source"].(map[string]interface{})
		score, _ := hit["_score"].(float64)
		results = append(results, SearchResult{
			ID:             getString(src, "id"),
			Type:           getString(src, "type"),
			Title:          getString(src, "title"),
			Description:    getString(src, "description"),
			URL:            getString(src, "url"),
			RelevanceScore: score,
		})
	}
	return results, totalHits, nil
}

func getString(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}
