package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"gorm.io/gorm"

	"study_plan_backend/db"
	"study_plan_backend/models"
)

type AIInvocationContext struct {
	UserID       uint
	JobType      string
	JobID        string
	Phase        string
	BatchIndex   int
	AgentAttempt int
}

type aiInvocationContextKey struct{}

func WithAIInvocationContext(ctx context.Context, value AIInvocationContext) context.Context {
	return context.WithValue(ctx, aiInvocationContextKey{}, value)
}

func WithAIInvocationStep(ctx context.Context, phase string, batchIndex, agentAttempt int) context.Context {
	value, _ := ctx.Value(aiInvocationContextKey{}).(AIInvocationContext)
	value.Phase, value.BatchIndex, value.AgentAttempt = phase, batchIndex, agentAttempt
	return WithAIInvocationContext(ctx, value)
}

type AIInvocationResult struct {
	Status           string
	HTTPStatus       int
	FinishReason     string
	Error            error
	ResponseChars    int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type AIInvocationHandle struct {
	database *gorm.DB
	id       uint
	started  time.Time
}

func BeginAIInvocation(ctx context.Context, cfg models.AIConfig, prompt string, maxTokens, providerAttempt int) (*AIInvocationHandle, error) {
	if db.DB == nil || !db.DB.Migrator().HasTable(&models.AIInvocationLog{}) {
		return &AIInvocationHandle{}, nil
	}
	traceBytes := make([]byte, 16)
	if _, err := rand.Read(traceBytes); err != nil {
		return nil, fmt.Errorf("create ai invocation trace id: %w", err)
	}
	metadata, _ := ctx.Value(aiInvocationContextKey{}).(AIInvocationContext)
	if strings.TrimSpace(metadata.JobType) == "" {
		metadata.JobType = "unscoped"
	}
	fingerprint := sha256.Sum256([]byte(prompt + "\x00" + fmt.Sprintf("%d", maxTokens)))
	started := time.Now().UTC()
	row := models.AIInvocationLog{
		TraceID: hex.EncodeToString(traceBytes), UserID: metadata.UserID, JobType: metadata.JobType, JobID: boundedAIText(metadata.JobID, 64),
		Phase: boundedAIText(metadata.Phase, 32), BatchIndex: metadata.BatchIndex, AgentAttempt: metadata.AgentAttempt, ProviderAttempt: providerAttempt,
		Provider: NormalizeAIProvider(cfg.Provider), ModelName: boundedAIText(cfg.ModelName, 128), RequestFingerprint: hex.EncodeToString(fingerprint[:]),
		PromptChars: len([]rune(prompt)), MaxTokens: maxTokens, Status: "started", StartedAt: started, RetainUntil: started.AddDate(0, 0, 90),
	}
	if err := db.DB.WithContext(context.Background()).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("persist ai invocation start: %w", err)
	}
	return &AIInvocationHandle{database: db.DB, id: row.ID, started: started}, nil
}

func (handle *AIInvocationHandle) Finish(result AIInvocationResult) error {
	if handle == nil || handle.database == nil || handle.id == 0 {
		return nil
	}
	finished := time.Now().UTC()
	status := strings.TrimSpace(result.Status)
	if status == "" {
		if result.Error == nil {
			status = "succeeded"
		} else {
			status = "failed"
		}
	}
	errorCode, errorMessage := classifyAIInvocationError(result.Error, result.HTTPStatus)
	updates := map[string]any{
		"status": status, "http_status": result.HTTPStatus, "finish_reason": boundedAIText(result.FinishReason, 32),
		"error_code": errorCode, "error_message": errorMessage, "response_chars": result.ResponseChars,
		"prompt_tokens": result.PromptTokens, "completion_tokens": result.CompletionTokens, "total_tokens": result.TotalTokens,
		"duration_ms": finished.Sub(handle.started).Milliseconds(), "finished_at": finished,
	}
	resultDB := handle.database.WithContext(context.Background()).Model(&models.AIInvocationLog{}).Where("id = ? AND status = ?", handle.id, "started").Updates(updates)
	if resultDB.Error != nil {
		return fmt.Errorf("finish ai invocation trace: %w", resultDB.Error)
	}
	if resultDB.RowsAffected != 1 {
		return errors.New("finish ai invocation trace: trace was already completed or missing")
	}
	return nil
}

func classifyAIInvocationError(err error, httpStatus int) (string, string) {
	if err == nil {
		return "", ""
	}
	code := "provider_error"
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		code = "timeout"
	case errors.Is(err, context.Canceled):
		code = "cancelled"
	case httpStatus == 429:
		code = "http_429"
	case httpStatus >= 500:
		code = "http_5xx"
	case httpStatus >= 400:
		code = "http_4xx"
	case IsAIOutputTruncated(err):
		code = "truncated"
	default:
		var networkError net.Error
		if errors.As(err, &networkError) {
			code = "network_error"
		} else if strings.Contains(strings.ToLower(err.Error()), "invalid completion") {
			code = "invalid_completion"
		} else if strings.Contains(strings.ToLower(err.Error()), "response exceeds") {
			code = "response_too_large"
		}
	}
	return code, boundedAIText(strings.ReplaceAll(err.Error(), "\n", " "), 256)
}

func boundedAIText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}
