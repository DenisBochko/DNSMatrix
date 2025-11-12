package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TaskMessage struct {
	ID             uuid.UUID         `json:"id"`                 // ID уникальный идентификатор задачи
	Target         string            `json:"target"`             // Target домен или IP, который нужно проверить
	TimeoutSeconds int               `json:"timeoutSeconds"`     // TimeoutSeconds время выполнения всех задачи в секундах
	ClientContext  ClientContext     `json:"clientContext"`      // ClientContext информация о клиенте, от которого инициирована проверка
	Checks         []CheckRequest    `json:"checks"`             // Checks список проверок
	Metadata       map[string]string `json:"metadata,omitempty"` // Metadata дополнительная информация
}

type ClientContext struct {
	IP        string `json:"ip,omitempty"`
	ASN       int    `json:"asn,omitempty"`
	Geo       Geo    `json:"geo,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

type Geo struct {
	Region    string `json:"region,omitempty"`
	Continent string `json:"continent,omitempty"`
}

type CheckRequest struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

type HTTPParams struct {
	Scheme              string            `json:"scheme"`
	Path                string            `json:"path"`
	Headers             map[string]string `json:"headers,omitempty"`
	ExpectedStatusRange [2]int            `json:"expectedStatusRange"`
	FollowRedirects     bool              `json:"followRedirects"`
	MaxBodyBytes        int               `json:"maxBodyBytes"`
}

type PingParams struct {
	Count      int `json:"count" example:"4"`
	IntervalMs int `json:"intervalMs" example:"1000"`
}

type TCPParams struct {
	Port             int `json:"port"`
	ConnectTimeoutMs int `json:"connectTimeoutMs"`
}

type TracerouteParams struct {
	Mode    string `json:"mode"`
	Port    int    `json:"port"`
	MaxHops int    `json:"maxHops"`
	Paris   bool   `json:"paris"`
}

type DNSParams struct {
	Records  []string `json:"records"`
	Resolver string   `json:"resolver,omitempty"`
}

type CheckResult struct {
	TaskID     uuid.UUID       `json:"taskId"`
	CheckIndex int             `json:"checkIndex"`
	Type       string          `json:"type"`
	Target     string          `json:"target"`
	StartedAt  time.Time       `json:"startedAt"`
	DurationMs int64           `json:"durationMs"`
	OK         bool            `json:"ok"`
	Error      string          `json:"error,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"` // разный по проверкам
}
