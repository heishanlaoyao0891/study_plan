package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

type planningTestProvider struct {
	generate func(context.Context, string, int) (string, error)
}

func (p planningTestProvider) Test() error { return nil }
func (p planningTestProvider) Generate(prompt string, tokens int) (string, error) {
	return p.GenerateContext(context.Background(), prompt, tokens)
}
func (p planningTestProvider) GenerateContext(ctx context.Context, prompt string, tokens int) (string, error) {
	return p.generate(ctx, prompt, tokens)
}

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

func TestGeneratePlanReturnsLocalPreviewForEnrichmentFailures(t *testing.T) {
	tests := []struct {
		name       string
		configure  func()
		status     string
		cancel     bool
		wantSource string
	}{
		{name: "disabled", configure: func() {
			currentAIProvider = func(context.Context) (models.AIConfig, services.AIProvider, error) {
				cfg := models.AIConfig{Provider: services.AIProviderSiliconFlow, Enabled: false, DailyGenerationLimit: 5}
				return cfg, planningTestProvider{}, nil
			}
		}, status: "disabled", wantSource: "local"},
		{name: "configuration error", configure: func() {
			currentAIProvider = validPlanningTestProvider(func(context.Context, string, int) (string, error) { return "", nil })
			validateAIConfigContext = func(context.Context, models.AIConfig, bool) error { return errors.New("bad config") }
		}, status: "configuration_error", wantSource: "local"},
		{name: "quota", configure: func() {
			currentAIProvider = validPlanningTestProvider(func(context.Context, string, int) (string, error) { return "", nil })
			canUseAIGeneration = func(context.Context, uint, int) (bool, int64, error) { return false, 5, nil }
		}, status: "quota_limited", wantSource: "local"},
		{name: "timeout", configure: func() {
			currentAIProvider = validPlanningTestProvider(func(context.Context, string, int) (string, error) { return "", context.DeadlineExceeded })
		}, status: "timeout", wantSource: "local"},
		{name: "invalid output", configure: func() {
			currentAIProvider = validPlanningTestProvider(func(context.Context, string, int) (string, error) { return "not json", nil })
		}, status: "invalid_output", wantSource: "local"},
		{name: "provider error", configure: func() {
			currentAIProvider = validPlanningTestProvider(func(context.Context, string, int) (string, error) { return "", errors.New("provider unavailable") })
		}, status: "provider_error", wantSource: "local"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupTestDB(t)
			originalCurrent, originalQuota, originalValidate := currentAIProvider, canUseAIGeneration, validateAIConfigContext
			t.Cleanup(func() {
				currentAIProvider, canUseAIGeneration, validateAIConfigContext = originalCurrent, originalQuota, originalValidate
			})
			validateAIConfigContext = func(context.Context, models.AIConfig, bool) error { return nil }
			canUseAIGeneration = func(context.Context, uint, int) (bool, int64, error) { return true, 0, nil }
			test.configure()
			router := gin.New()
			router.POST("/generate", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uint(42)) }, GeneratePlan)
			body, _ := json.Marshal(gin.H{"goal": "Study Go", "days": 1, "hours_per_day": 1, "start_date": "2026-08-01", "available_time_slot": "20:00-21:00"})
			request := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			if test.cancel {
				ctx, cancel := context.WithCancel(request.Context())
				cancel()
				request = request.WithContext(ctx)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("expected local success, got %d: %s", recorder.Code, recorder.Body.String())
			}
			var response struct {
				Data struct {
					Source     string               `json:"source"`
					Preview    services.PlanPreview `json:"preview"`
					Enrichment struct {
						Status string `json:"status"`
					} `json:"enrichment"`
				} `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Data.Source != test.wantSource || response.Data.Enrichment.Status != test.status || len(response.Data.Preview.Tasks) != 1 {
				t.Fatalf("unexpected fallback response: %+v", response.Data)
			}
		})
	}
}

func TestGeneratePlanStopsWhenRequestIsCancelled(t *testing.T) {
	setupTestDB(t)
	router := gin.New()
	router.POST("/generate", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uint(42)) }, GeneratePlan)
	request := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(`{"goal":"Study Go","days":1}`))
	request.Header.Set("Content-Type", "application/json")
	requestContext, cancel := context.WithCancel(request.Context())
	cancel()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request.WithContext(requestContext))
	if responseCode(t, recorder.Body.Bytes()) != http.StatusRequestTimeout {
		t.Fatalf("expected cancelled context collection to stop: %s", recorder.Body.String())
	}
}

func validPlanningTestProvider(generate func(context.Context, string, int) (string, error)) func(context.Context) (models.AIConfig, services.AIProvider, error) {
	return func(context.Context) (models.AIConfig, services.AIProvider, error) {
		cfg := models.AIConfig{Provider: services.AIProviderSiliconFlow, Enabled: true, DailyGenerationLimit: 5, RequestTimeoutSeconds: 30}
		return cfg, planningTestProvider{generate: generate}, nil
	}
}

func TestCommitAIPlanRecomputesPersistsSourceAndIsIdempotent(t *testing.T) {
	setupTestDB(t)
	uid := uint(77)
	original := services.PlanPreview{Title: "Study", Summary: "Summary", Tasks: []services.PlanPreviewTask{
		{Date: "2026-08-02", PlannedStart: "20:00", PlannedEnd: "20:30", Title: "Second", Objective: "complete second exercise", EstimatedMinutes: 30},
		{Date: "2026-08-01", PlannedStart: "19:00", PlannedEnd: "20:00", Title: "First", Objective: "complete first exercise", EstimatedMinutes: 60},
	}}
	mustAssignTaskIdentities(t, &original)
	token, err := services.SignPlanProvenance(uid, "local", original)
	if err != nil {
		t.Fatal(err)
	}
	edited := original
	edited.Tasks = append([]services.PlanPreviewTask(nil), original.Tasks...)
	edited.EstimatedTotalHours = 999
	edited.Tasks[0].EstimatedMinutes = 999
	body := gin.H{"preview": edited, "provenance_token": token, "idempotency_key": previewCommitKey("commit", original)}
	first := callJSONHandler(t, CommitAIPlan, uid, "/commit", "", body)
	if responseCode(t, first) != 0 {
		t.Fatalf("commit failed: %s", first)
	}
	second := callJSONHandler(t, CommitAIPlan, uid, "/commit", "", body)
	if responseCode(t, second) != 0 {
		t.Fatalf("idempotent replay failed: %s", second)
	}
	var plans []models.Plan
	if err := db.DB.Where("user_id = ?", uid).Find(&plans).Error; err != nil || len(plans) != 1 {
		t.Fatalf("expected one plan after retry: plans=%+v err=%v", plans, err)
	}
	var tasks []models.DailyTask
	db.DB.Where("plan_id = ?", plans[0].ID).Order("sort_order ASC").Find(&tasks)
	if plans[0].GenerationSource != "local" || !plans[0].AIGenerated || plans[0].StartDate != "2026-08-01" || plans[0].EndDate != "2026-08-02" || plans[0].WeeklyTargetHours != 2 || len(tasks) != 2 || tasks[0].Date != "2026-08-01" || tasks[1].EstimatedMinutes != 30 {
		t.Fatalf("commit did not recompute authoritative fields: plan=%+v tasks=%+v", plans[0], tasks)
	}
}

func TestCommitAIPlanRequiresTrustedProvenance(t *testing.T) {
	setupTestDB(t)
	preview := services.PlanPreview{Title: "Study", Tasks: []services.PlanPreviewTask{{Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "21:00", Title: "Task", Objective: "complete one exercise", EstimatedMinutes: 60}}}
	body := gin.H{"preview": preview, "provenance_token": "forged", "idempotency_key": "commit_key_1234567890"}
	response := callJSONHandler(t, CommitAIPlan, 88, "/commit", "", body)
	if responseCode(t, response) != http.StatusBadRequest {
		t.Fatalf("forged source provenance was accepted: %s", response)
	}
}

func TestAIRequestBodiesAreBounded(t *testing.T) {
	setupTestDB(t)
	router := gin.New()
	router.POST("/generate", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uint(1)) }, GeneratePlan)
	request := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(`{"goal":"`+strings.Repeat("x", maxAIGenerateBodyBytes)+`"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if responseCode(t, recorder.Body.Bytes()) != http.StatusBadRequest {
		t.Fatalf("oversized generation body was accepted: %s", recorder.Body.String())
	}
}

func TestCommitAIPlanAppliesOverloadConfirmation(t *testing.T) {
	setupTestDB(t)
	uid := uint(99)
	for index := 0; index < maxActivePlans; index++ {
		db.DB.Create(&models.Plan{UserID: uid, Title: "Existing", Status: models.PlanStatusActive})
	}
	preview := services.PlanPreview{Title: "Study", Tasks: []services.PlanPreviewTask{{Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "21:00", Title: "Task", Objective: "complete one exercise", EstimatedMinutes: 60}}}
	mustAssignTaskIdentities(t, &preview)
	token, err := services.SignPlanProvenance(uid, "local", preview)
	if err != nil {
		t.Fatal(err)
	}
	body := gin.H{"preview": preview, "provenance_token": token, "idempotency_key": previewCommitKey("overload", preview)}
	if response := callJSONHandler(t, CommitAIPlan, uid, "/commit", "", body); responseCode(t, response) != http.StatusBadRequest {
		t.Fatalf("unconfirmed overload was accepted: %s", response)
	}
	body["confirm_overload"] = true
	if response := callJSONHandler(t, CommitAIPlan, uid, "/commit", "", body); responseCode(t, response) != 0 {
		t.Fatalf("confirmed overload was rejected: %s", response)
	}
}

func TestCommitAIPlanRejectsUnrelatedPreviewSubstitution(t *testing.T) {
	setupTestDB(t)
	uid := uint(101)
	preview := services.PlanPreview{Title: "Study", Summary: "Bound summary", Tasks: []services.PlanPreviewTask{
		{Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "21:00", Title: "First", Objective: "complete first exercise"},
		{Date: "2026-08-02", PlannedStart: "20:00", PlannedEnd: "21:00", Title: "Second", Objective: "complete second exercise"},
	}}
	mustAssignTaskIdentities(t, &preview)
	token, err := services.SignPlanProvenance(uid, "local", preview)
	if err != nil {
		t.Fatal(err)
	}
	substituted := preview
	substituted.Tasks = append([]services.PlanPreviewTask(nil), preview.Tasks...)
	substituted.Tasks[0].Identity = substituted.Tasks[1].Identity
	body := gin.H{"preview": substituted, "provenance_token": token, "idempotency_key": previewCommitKey("substitution", preview)}
	if response := callJSONHandler(t, CommitAIPlan, uid, "/commit", "", body); responseCode(t, response) != http.StatusBadRequest {
		t.Fatalf("unrelated task substitution was accepted: %s", response)
	}
	removed := preview
	removed.Tasks = removed.Tasks[:1]
	body["preview"] = removed
	if response := callJSONHandler(t, CommitAIPlan, uid, "/commit", "", body); responseCode(t, response) != http.StatusBadRequest {
		t.Fatalf("task removal was accepted: %s", response)
	}
}

func TestCommitAIPlanConcurrentIdempotency(t *testing.T) {
	setupTestDB(t)
	uid := uint(102)
	preview := services.PlanPreview{Title: "Study", Tasks: []services.PlanPreviewTask{{Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "21:00", Title: "Task", Objective: "complete one exercise"}}}
	mustAssignTaskIdentities(t, &preview)
	token, err := services.SignPlanProvenance(uid, "local", preview)
	if err != nil {
		t.Fatal(err)
	}
	body := gin.H{"preview": preview, "provenance_token": token, "idempotency_key": previewCommitKey("concurrent", preview)}
	responses := make([][]byte, 8)
	var wait sync.WaitGroup
	for index := range responses {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			responses[index] = callJSONHandler(t, CommitAIPlan, uid, "/commit", "", body)
		}(index)
	}
	wait.Wait()
	for _, response := range responses {
		if responseCode(t, response) != 0 {
			t.Fatalf("concurrent same-payload replay failed: %s", response)
		}
	}
	var planCount int64
	db.DB.Model(&models.Plan{}).Where("user_id = ?", uid).Count(&planCount)
	if planCount != 1 {
		t.Fatalf("expected one plan, got %d", planCount)
	}
	different := preview
	different.Tasks = append([]services.PlanPreviewTask(nil), preview.Tasks...)
	different.Tasks[0].Title = "Different semantic edit"
	body["preview"] = different
	if response := callJSONHandler(t, CommitAIPlan, uid, "/commit", "", body); responseCode(t, response) != http.StatusConflict {
		t.Fatalf("same key with different payload did not conflict: %s", response)
	}
}

func mustAssignTaskIdentities(t *testing.T, preview *services.PlanPreview) {
	t.Helper()
	if err := services.AssignPlanTaskIdentities(preview); err != nil {
		t.Fatal(err)
	}
}

func previewCommitKey(prefix string, preview services.PlanPreview) string {
	return prefix + "_key_" + preview.Tasks[0].Identity
}
