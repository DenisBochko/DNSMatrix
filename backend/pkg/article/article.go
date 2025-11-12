package elasticsearch

import (
	"context"
	"fmt"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v9"
)

const (
	DefaultTimeout = 10 * time.Second
)

type Elasticsearch interface {
	Client() *elasticsearch.Client
	Ping(ctx context.Context) error
}

type Config struct {
	Addresses []string
	Username  string
	Password  string
	CloudID   string // If using Elastic Cloud
	APIKey    string // If using API Key
	Timeout   time.Duration
}

type elasticsearchClient struct {
	client  *elasticsearch.Client
	timeout time.Duration
}

func New(cfg *Config) (Elasticsearch, error) {
	if len(cfg.Addresses) == 0 && cfg.CloudID == "" {
		return nil, fmt.Errorf("no Elasticsearch addresses or CloudID provided")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}

	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		CloudID:   cfg.CloudID,
		APIKey:    cfg.APIKey,
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	es := &elasticsearchClient{
		client:  client,
		timeout: cfg.Timeout,
	}

	if err := es.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping elasticsearch: %w", err)
	}

	return es, nil
}

func (e *elasticsearchClient) Client() *elasticsearch.Client {
	return e.client
}

func (e *elasticsearchClient) Ping(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	res, err := e.client.Ping(e.client.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("ping request failed: %w", err)
	}

	defer func() {
		if cErr := res.Body.Close(); cErr != nil {
			err = fmt.Errorf("%w, failed to close response body: %w", err, cErr)
		}
	}()

	if res.IsError() {
		return fmt.Errorf("elasticsearch ping returned error: %s", res.String())
	}

	return nil
}
