package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"telegram-analytics-api/internal/domain"
	"telegram-analytics-api/internal/service"
	"telegram-analytics-api/internal/telegram"
)

type mockTelegramClient struct {
	chatInfo  domain.ChatInfo
	messages  []domain.Message
	err       error
	sendMsgID int
	sendErr   error
}

func (m *mockTelegramClient) SendMessage(ctx context.Context, chatID, text string) (int, error) {
	return m.sendMsgID, m.sendErr
}
func (m *mockTelegramClient) GetChatInfo(ctx context.Context, chatID string) (domain.ChatInfo, error) {
	return m.chatInfo, m.err
}
func (m *mockTelegramClient) IterateMessages(ctx context.Context, chatID string, since time.Time, limit int) ([]domain.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	var filtered []domain.Message
	for _, msg := range m.messages {
		if since.IsZero() || msg.Date.After(since) || msg.Date.Equal(since) {
			filtered = append(filtered, msg)
		}
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}
func (m *mockTelegramClient) Start(ctx context.Context) error { return nil }
func (m *mockTelegramClient) Stop(ctx context.Context) error  { return nil }
func (m *mockTelegramClient) WaitReady() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func setupRouterWithMock(mock telegram.ClientInterface) *gin.Engine {
	logger, _ := zap.NewDevelopment()
	svc := service.NewAnalyticsService(mock)
	handler := NewHandler(svc)
	return SetupRouter(handler, logger)
}

func TestHandler_SendMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name         string
		requestBody  map[string]interface{}
		mockID       int
		mockErr      error
		wantStatus   int
		wantResponse map[string]interface{}
	}{
		{
			name: "success",
			requestBody: map[string]interface{}{
				"chat_id": "@test",
				"text":    "hello",
			},
			mockID:     123,
			mockErr:    nil,
			wantStatus: http.StatusOK,
			wantResponse: map[string]interface{}{
				"message_id": float64(123),
				"status":     "sent",
			},
		},
		{
			name: "bad request - missing chat_id",
			requestBody: map[string]interface{}{
				"text": "hello",
			},
			wantStatus:   http.StatusBadRequest,
			wantResponse: map[string]interface{}{"detail": "Key: 'SendMessageRequest.ChatID' Error:Field validation for 'ChatID' failed on the 'required' tag"},
		},
		{
			name: "telegram error",
			requestBody: map[string]interface{}{
				"chat_id": "@test",
				"text":    "hello",
			},
			mockErr:      assert.AnError,
			wantStatus:   http.StatusBadRequest,
			wantResponse: map[string]interface{}{"detail": assert.AnError.Error()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTelegramClient{
				sendMsgID: tt.mockID,
				sendErr:   tt.mockErr,
			}
			router := setupRouterWithMock(mock)
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/send_message/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.wantResponse, resp)
		})
	}
}

func TestHandler_GetSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name         string
		query        string
		mockChatInfo domain.ChatInfo
		mockMessages []domain.Message
		mockErr      error
		wantStatus   int
		checkFunc    func(t *testing.T, resp map[string]interface{})
	}{
		{
			name:  "success with keyword",
			query: "?group_id=@test&keyword=COLLABA25",
			mockChatInfo: domain.ChatInfo{
				Subscribers: 1000,
				Description: "Test COLLABA25",
			},
			mockMessages: []domain.Message{
				{ID: 1, Date: time.Now().Add(-5 * time.Hour), Views: 200, Reactions: map[string]int{"👍": 10}, Text: "Hello COLLABA25"},
				{ID: 2, Date: time.Now().Add(-10 * time.Hour), Views: 150, Reactions: map[string]int{"❤️": 5}, Text: "No keyword here"},
			},
			wantStatus: http.StatusOK,
			checkFunc: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, true, resp["contains_keyword"])
				assert.Equal(t, 1000.0, resp["subscribers"])
				assert.Equal(t, 200.0, resp["average_views"])
				assert.Equal(t, 10.0, resp["average_reactions"])
				assert.Equal(t, 5.0, resp["engagement_rate_percent"]) // 10/200*100 = 5
				assert.Equal(t, 20.0, resp["activity_percentage"])    // 200/1000*100 = 20
			},
		},
		{
			name:  "success without keyword",
			query: "?group_id=@test",
			mockChatInfo: domain.ChatInfo{
				Subscribers: 1000,
				Description: "No keyword",
			},
			mockMessages: []domain.Message{
				{ID: 1, Date: time.Now().Add(-5 * time.Hour), Views: 200},
			},
			wantStatus: http.StatusOK,
			checkFunc: func(t *testing.T, resp map[string]interface{}) {
				_, ok := resp["contains_keyword"]
				assert.False(t, ok, "contains_keyword should be omitted")
			},
		},
		{
			name:       "missing group_id",
			query:      "",
			wantStatus: http.StatusBadRequest,
			checkFunc: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, "missing group_id", resp["detail"])
			},
		},
		{
			name:       "telegram error",
			query:      "?group_id=@test",
			mockErr:    assert.AnError,
			wantStatus: http.StatusBadRequest,
			checkFunc: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, assert.AnError.Error(), resp["detail"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTelegramClient{
				chatInfo: tt.mockChatInfo,
				messages: tt.mockMessages,
				err:      tt.mockErr,
			}
			router := setupRouterWithMock(mock)
			req := httptest.NewRequest(http.MethodGet, "/get/"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, resp)
			}
		})
	}
}
