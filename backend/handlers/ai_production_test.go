package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"study_plan_backend/middleware"
)

func TestGeneratePlanReportsRuleFallbackTruthfully(t *testing.T) {
	setupTestDB(t)
	router := gin.New()
	router.POST("/generate", func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(42))
	}, GeneratePlan)

	body, _ := json.Marshal(gin.H{"goal": "Study Go", "days": 1, "hours_per_day": 1, "start_date": "2026-08-01", "available_time_slot": "20:00-21:00"})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected fallback generation success, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Mode     string `json:"mode"`
			AIStatus struct {
				Mode           string `json:"mode"`
				Provider       string `json:"provider"`
				FallbackReason string `json:"fallback_reason"`
			} `json:"ai_status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Mode != "fallback" || response.Data.AIStatus.Mode != "fallback" || response.Data.AIStatus.Provider != "mock" || response.Data.AIStatus.FallbackReason == "" {
		t.Fatalf("fallback source was not reported truthfully: %+v", response.Data)
	}
}
