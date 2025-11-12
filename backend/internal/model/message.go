package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TaskMessageRequest представляет задачу для агента
// @Description Task для выполнения проверок сети (HTTP, ping, TCP, traceroute, DNS)
type TaskMessageRequest struct {
	Target         string                `binding:"required" json:"target" example:"example.com"` // Target домен или IP, который нужно проверить
	TimeoutSeconds int                   `binding:"required" json:"timeoutSeconds" example:"20"`  // TimeoutSeconds время выполнения всех задачи в секундах
	Broadcast      bool                  `json:"broadcast" example:"false"`                       // Отправлять ли запрос на агенты всех регионов или берётся ближайший 1 агент к клиенту
	Checks         []CheckRequestRequest `binding:"required" json:"checks"`                       // Checks список проверок
} // @Name TaskMessageRequest

// CheckRequestRequest
// @Description описание одной проверки
type CheckRequestRequest struct {
	Type   string                 `binding:"required" json:"type" example:"http"` // Type тип проверки: http|ping|tcp|traceroute|dns
	Params map[string]interface{} `binding:"required" json:"params"`              // Params параметры проверки
} // @Name CheckRequestRequest

// HTTPParamsRequest
// @Description параметры http-проверки
type HTTPParamsRequest struct {
	Scheme              string            `binding:"required" json:"scheme" example:"https"`
	Path                string            `binding:"required" json:"path" example:"/health, /"`
	Headers             map[string]string `binding:"required" json:"headers,omitempty"`
	ExpectedStatusRange [2]int            `binding:"required" json:"expectedStatusRange" example:"[200,299]"`
	FollowRedirects     bool              `binding:"required" json:"followRedirects" example:"true"`
	MaxBodyBytes        int               `binding:"required" json:"maxBodyBytes" example:"4096"`
} // @Name HTTPParamsRequest

// PingParamsRequest
// @Description параметры ping
type PingParamsRequest struct {
	Count      int `binding:"required" json:"count" example:"4"`
	IntervalMs int `binding:"required" json:"intervalMs" example:"1000"`
} // @Name PingParamsRequest

// TCPParamsRequest
// @Description параметры TCP
type TCPParamsRequest struct {
	Port             int `binding:"required" json:"port" example:"443"`
	ConnectTimeoutMs int `binding:"required" json:"connectTimeoutMs" example:"3000"`
} // @Name TCPParamsRequest

// TracerouteParamsRequest
// @Description параметры traceroute
type TracerouteParamsRequest struct {
	Mode    string `binding:"required" json:"mode" example:"tcp"`
	Port    int    `binding:"required" json:"port" example:"443"`
	MaxHops int    `binding:"required" json:"maxHops" example:"30"`
	Paris   bool   `binding:"required" json:"paris" example:"true"`
} // @Name TracerouteParamsRequest

// DNSParamsRequest
// @Description параметры DNS
type DNSParamsRequest struct {
	Records  []string `binding:"required" json:"records" example:"[\"A\",\"AAAA\",\"MX\"]"`
	Resolver string   `binding:"required" json:"resolver,omitempty" example:"8.8.8.8"`
} // @Name DNSParamsRequest

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
}

type DNSParams struct {
	Records  []string `json:"records"`
	Resolver string   `json:"resolver,omitempty"`
}

type Request struct {
	ID             uuid.UUID `db:"id" json:"id"`
	Target         string    `db:"target" json:"target"`
	TimeoutSeconds int       `db:"timeout_seconds" json:"timeoutSeconds"`
	Broadcast      bool      `db:"broadcast" json:"broadcast"`
	ClientIP       string    `db:"client_ip" json:"clientIP"`
	UserAgent      string    `db:"user_agent" json:"userAgent"`
	ClientASN      int       `db:"client_asn" json:"clientASN"`
	ClientCC       string    `db:"client_cc" json:"clientCC"`
	ClientRegion   string    `db:"client_region" json:"clientRegion"`
	Status         string    `db:"status" json:"status"`
	ChecksTypes    []string  `db:"checks_types" json:"checkTypes"`
	RequestJSON    []byte    `db:"request_json" json:"requestJSON"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}

type Assignment struct {
	ID          uuid.UUID `db:"id" json:"id"`
	RequestID   uuid.UUID `db:"request_id" json:"requestId"`
	AgentID     uuid.UUID `db:"agent_id" json:"agentId"`
	AgentRegion string    `db:"agent_region" json:"agentRegion"`
	Status      string    `db:"status" json:"status"`
	EnqueuedAt  time.Time `db:"enqueued_at" json:"enqueuedAt"`
	StartedAt   time.Time `db:"started_at" json:"startedAt"`
	FinishedAt  time.Time `db:"finished_at" json:"finishedAt"`
	ErrorText   string    `db:"error_text" json:"errorText"`
	OutboxID    uuid.UUID `db:"outbox_id" json:"outboxId"`
}

type CheckResult struct {
	ID           uuid.UUID `db:"id" json:"id"`
	AssignmentId uuid.UUID `db:"assignment_id" json:"assignmentId"`
	Type         string    `db:"type" json:"type"`
	Status       string    `db:"status" json:"status"`
	StartedAt    time.Time `db:"started_at" json:"startedAt"`
	FinishedAt   time.Time `db:"finished_at" json:"finishedAt"`
	Payload      []byte    `db:"payload" json:"payload"`
}

type CheckResultFromAgent struct {
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

type CheckResultResponse struct {
	RequestID   uuid.UUID       `db:"request_id" json:"requestId" `
	AgentID     uuid.UUID       `db:"agent_id" json:"agentId"`
	AgentRegion string          `db:"agent_region" json:"agentRegion"`
	Type        string          `db:"type" json:"type"`
	Status      string          `db:"status" json:"status"`
	StartedAt   time.Time       `db:"started_at" json:"startedAt"`
	FinishedAt  time.Time       `db:"finished_at" json:"finishedAt"`
	Payload     json.RawMessage `db:"payload" json:"payload,omitempty"`
}

type RequestIDPathParam struct {
	ID string `uri:"request_id" binding:"required,uuid" example:"b4b03119-1290-44bc-b599-6a5e91d6611f"`
}

/*
{
  "id": "task-20251025-0001",
  "target": "example.com",
  "timeout_seconds": 20,
  "client_context": {
    "ip": "203.0.113.57",
    "asn": 12345,
    "geo": {"country": "FI", "city":"Helsinki"},
    "user_agent": "Mozilla/5.0"
  },
  "assigned_agent": "agent-eu-ams-03",           // optional: orchestrator fills
  "checks": [
    {
      "type": "http",
      "params": {
        "scheme": "https",
        "path": "/",
        "headers": {"User-Agent":"agent-probe/1.0"},
        "expected_status_range": [200,299],
        "follow_redirects": true,
        "max_body_bytes": 4096
      }
    },
    {
      "type": "ping",
      "params": {"count": 4, "interval_ms": 1000}
    },
    {
      "type": "tcp",
      "params": {"port": 443, "connect_timeout_ms": 3000}
    },
    {
      "type": "traceroute",
      "params": {"mode": "tcp", "port": 443, "max_hops": 30, "paris": true}
    },
    {
      "type": "dns",
      "params": {"records": ["A","AAAA","MX","NS","TXT"], "resolver": "8.8.8.8"}
    }
  ],
  "metadata": {"origin": "region-eu-1", "requester": "scheduler"}
}
*/
