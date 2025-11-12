package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v9"

	"hackathon-back/internal/apperrors"
	"hackathon-back/internal/model"
)

const indexName = "articles"

type ElasticRepo struct {
	es *elasticsearch.Client
}

func NewElasticRepository(es *elasticsearch.Client) *ElasticRepo {
	return &ElasticRepo{es: es}
}

func (r *ElasticRepo) EnsureIndex(ctx context.Context) (err error) {
	exists, err := r.es.Indices.Exists([]string{indexName}, r.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("check index existence: %w", err)
	}

	defer func() {
		if cErr := exists.Body.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close response body: %w", err, cErr)
		}
	}()

	if exists.StatusCode == http.StatusOK {
		return nil
	}

	if exists.StatusCode != http.StatusNotFound && exists.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status on exists: %s", exists.Status())
	}

	mapping := `{
		"settings": {
			"analysis": {
				"analyzer": {
					"ru_text": { "type": "russian" },
					"en_text": { "type": "english" }
				}
			}
		},
		"mappings": {
			"properties": {
				"title_ru":   { "type": "text", "analyzer": "ru_text" },
				"title_en":   { "type": "text", "analyzer": "en_text" },
				"content_ru": { "type": "text", "analyzer": "ru_text" },
				"content_en": { "type": "text", "analyzer": "en_text" },
				"created_at": { "type": "date" },
				"updated_at": { "type": "date" }
			}
		}
	}`

	res, err := r.es.Indices.Create(indexName, r.es.Indices.Create.WithBody(strings.NewReader(mapping)), r.es.Indices.Create.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	defer func() {
		if cErr := exists.Body.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close response body: %w", err, cErr)
		}
	}()

	if res.IsError() {
		return fmt.Errorf("index creation failed: %s", res.String())
	}

	_, err = r.es.Cluster.Health(
		r.es.Cluster.Health.WithContext(ctx),
		r.es.Cluster.Health.WithWaitForStatus("yellow"),
		r.es.Cluster.Health.WithTimeout(10*time.Second),
	)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

func (r *ElasticRepo) Create(ctx context.Context, article *model.Article) (err error) {
	data, err := json.Marshal(article)
	if err != nil {
		return fmt.Errorf("failed to marshal article: %w", err)
	}

	res, err := r.es.Index(
		indexName,
		bytes.NewReader(data),
		r.es.Index.WithDocumentID(article.ID.String()),
		r.es.Index.WithContext(ctx),
	)
	if err != nil {
		return err
	}

	defer func() {
		if cErr := res.Body.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close response body: %w", err, cErr)
		}
	}()

	if res.IsError() {
		return fmt.Errorf("failed to create article: %s", res.String())
	}

	return nil
}

func (r *ElasticRepo) Get(ctx context.Context, id string) (article *model.Article, err error) {
	res, err := r.es.Get(indexName, id, r.es.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	defer func() {
		if cErr := res.Body.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close response body: %w", err, cErr)
		}
	}()

	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, apperrors.ErrArticleDoesNotExist
	default:
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("es get failed: %s: %s", res.Status(), string(b))
	}

	var doc struct {
		Source model.Article `json:"_source"`
	}

	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &doc.Source, nil
}

func (r *ElasticRepo) Delete(ctx context.Context, id string) (err error) {
	res, err := r.es.Delete(indexName, id, r.es.Delete.WithContext(ctx))
	if err != nil {
		return err
	}

	defer func() {
		if cErr := res.Body.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close response body: %w", err, cErr)
		}
	}()

	if res.StatusCode == http.StatusNotFound {
		return apperrors.ErrArticleDoesNotExist
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete article: %s", res.String())
	}

	return nil
}

func (r *ElasticRepo) Patch(ctx context.Context, id string, fields map[string]interface{}) (err error) {
	fields["updated_at"] = time.Now()

	payload := map[string]interface{}{"doc": fields}

	buf := new(bytes.Buffer)

	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return fmt.Errorf("encode doc: %w", err)
	}

	res, err := r.es.Update(indexName, id, buf, r.es.Update.WithContext(ctx))
	if err != nil {
		return err
	}

	defer func() {
		if cErr := res.Body.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close response body: %w", err, cErr)
		}
	}()

	if res.StatusCode == http.StatusNotFound {
		return apperrors.ErrArticleDoesNotExist
	}

	if res.IsError() {
		return fmt.Errorf("failed to patch article: %s", res.String())
	}

	return nil
}

func (r *ElasticRepo) Search(ctx context.Context, query string, from, size int, sort string) (results []model.SearchResult, err error) {
	type multiMatch struct {
		Query  string   `json:"query"`
		Fields []string `json:"fields"`
	}

	type bodyT struct {
		Query struct {
			MultiMatch multiMatch `json:"multi_match"`
		} `json:"query"`
		Highlight struct {
			PreTags  []string               `json:"pre_tags"`
			PostTags []string               `json:"post_tags"`
			Fields   map[string]interface{} `json:"fields"`
		} `json:"highlight"`
		TrackTotalHits bool          `json:"track_total_hits"`
		From           int           `json:"from,omitempty"`
		Size           int           `json:"size,omitempty"`
		Sort           []interface{} `json:"sort,omitempty"`
	}

	body := bodyT{}
	body.Query.MultiMatch = multiMatch{
		Query:  query,
		Fields: []string{"title_ru", "title_en", "content_ru", "content_en"},
	}
	body.Highlight.PreTags = []string{"<em>"}
	body.Highlight.PostTags = []string{"</em>"}
	body.Highlight.Fields = map[string]interface{}{
		"title_ru": struct{}{}, "title_en": struct{}{},
		"content_ru": struct{}{}, "content_en": struct{}{},
	}

	body.TrackTotalHits = true
	if from > 0 {
		body.From = from
	}

	if size > 0 {
		body.Size = size
	}

	if sort != "" {
		// "created_at:desc" -> [{"created_at":{"order":"desc"}}]
		order := "asc"
		field := sort

		if i := strings.IndexByte(sort, ':'); i > 0 {
			field, order = sort[:i], sort[i+1:]
		}

		body.Sort = []interface{}{
			map[string]interface{}{field: map[string]string{"order": order}},
		}
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(&body); err != nil {
		return nil, fmt.Errorf("encode search body: %w", err)
	}

	res, err := r.es.Search(
		r.es.Search.WithContext(ctx),
		r.es.Search.WithIndex(indexName),
		r.es.Search.WithBody(buf),
		r.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		if cErr := res.Body.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close response body: %w", err, cErr)
		}
	}()

	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.String())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source    model.Article       `json:"_source"`
				Highlight map[string][]string `json:"highlight,omitempty"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	out := make([]model.SearchResult, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		out = append(out, model.SearchResult{
			Article:   hit.Source,
			Highlight: hit.Highlight,
		})
	}

	return out, nil
}
