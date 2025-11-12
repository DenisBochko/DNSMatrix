package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"hackathon-back/internal/model"
)

type RequestService interface {
	CreateRequest(ctx context.Context, req model.TaskMessageRequest, ip net.IP, ua string) (*model.Request, error)
	GetResultsByRequestID(ctx context.Context, requestID uuid.UUID) ([]model.CheckResultResponse, error)
}

type RequestHandler struct {
	log *zap.Logger
	svc RequestService
}

func NewRequestHandler(log *zap.Logger, svc RequestService) *RequestHandler {
	return &RequestHandler{
		log: log,
		svc: svc,
	}
}

// Response helpers assumed to exist in this package:
//   - ResponseWithData
//   - ResponseWithMessage
//   - StatusErr, StatusSuccess
//   - UserAgentHeader

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// TODO: ограничь домены по необходимости
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsMessage struct {
	Type string      `json:"type"           ` // "snapshot" | "update" | "done" | "error"
	Data interface{} `json:"data,omitempty" ` // payload
	Err  string      `json:"error,omitempty"`
}

// CreateRequest
// @Summary Создать задачу сетевых проверок.
// @Description Принимает проверки, достаёт IP клиента, определяет регион, пишет в checks=PENDING и в outbox кладёт.
// @Tags Checks
// @Accept json
// @Produce json
// @Param payload body model.TaskMessageRequest true "Task payload"
// @Success 201 {object} ResponseWithData{data=model.Request} "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid JSON body"
// @Failure 500 {object} ResponseWithMessage "Failed to create request"
// @Router /check/task [post]
func (h *RequestHandler) CreateRequest(c *gin.Context) {
	ctx := c.Request.Context()

	var req model.TaskMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	clientIPStr := c.ClientIP()
	clientIP := net.ParseIP(clientIPStr)
	userAgent := c.GetHeader(UserAgentHeader)

	request, err := h.svc.CreateRequest(ctx, req, clientIP, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, ResponseWithData{
		Status: StatusSuccess,
		Data:   request,
	})
}

// GetResults
// @Summary Получить результаты сетевых проверок.
// @Description Возвращает текущее состояние проверок по request_id одним ответом.
// @Tags Checks
// @Produce json
// @Param id path string true "Request UUID"
// @Success 200 {object} ResponseWithData{data=[]model.CheckResultResponse} "Success"
// @Failure 400 {object} ResponseWithMessage "Invalid path param"
// @Failure 500 {object} ResponseWithMessage "Failed to get results"
// @Router /check/{id} [get]
func (h *RequestHandler) GetResults(c *gin.Context) {
	ctx := c.Request.Context()

	var uri model.RequestIDPathParam
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	requestUID, err := uuid.Parse(uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	res, err := h.svc.GetResultsByRequestID(ctx, requestUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ResponseWithMessage{
			Status:  StatusErr,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseWithData{
		Status: StatusSuccess,
		Data:   res,
	})
}

// StreamResults
// @Summary Стрим результатов сетевых проверок по WebSocket.
// @Description Открывает WS и присылает актуальные результаты по request_id до завершения всех проверок.
// @Tags Checks
// @Param request_id path string true "Request UUID"
// @Produce application/json
// @Router /check/ws/check/{request_id} [get]
func (h *RequestHandler) StreamResults(c *gin.Context) {
	// апгрейд в WS
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Warn("ws upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	// request_id
	requestIDStr := c.Param("request_id")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		_ = conn.WriteJSON(wsMessage{Type: "error", Err: "invalid request_id"})
		return
	}

	// keepalive
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// читаем входящие сообщения, чтобы не завис ping/pong
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ctx := c.Request.Context()
	ticker := time.NewTicker(1 * time.Second) // период опроса svc
	defer ticker.Stop()

	var lastHash string

	send := func(msg wsMessage) bool {
		if err := conn.WriteJSON(msg); err != nil {
			h.log.Warn("ws write failed", zap.Error(err))
			return false
		}
		return true
	}

	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteJSON(wsMessage{Type: "done"})
			return
		case <-ticker.C:
			results, err := h.svc.GetResultsByRequestID(ctx, requestID)
			if err != nil {
				if !send(wsMessage{Type: "error", Err: err.Error()}) {
					return
				}
				continue
			}

			// хэш снапшота, чтобы не слать дубликаты
			raw, _ := json.Marshal(results)
			sum := sha256.Sum256(raw)
			newHash := hex.EncodeToString(sum[:])

			if lastHash == "" {
				if !send(wsMessage{Type: "snapshot", Data: results}) {
					return
				}
				lastHash = newHash
			} else if newHash != lastHash {
				if !send(wsMessage{Type: "update", Data: results}) {
					return
				}
				lastHash = newHash
			}

			// закрываем при терминальном состоянии всех проверок
			if allTerminal(results) {
				_ = conn.WriteJSON(wsMessage{Type: "done", Data: results})
				return
			}

			// поддерживаем соединение живым
			_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
		}
	}
}

// Подстрой под свои статусы в model.CheckResultResponse.Status
func allTerminal(list []model.CheckResultResponse) bool {
	if len(list) == 0 {
		return false
	}
	for _, it := range list {
		switch it.Status {
		case "SUCCESS", "FAILED", "TIMEOUT", "CANCELLED":
			// терминальные
		default:
			return false
		}
	}
	return true
}
