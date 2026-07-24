package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/models"
)

type AIProvider interface {
	Test() error
	Generate(prompt string, maxTokens int) (string, error)
}

func NewAIProvider(cfg models.AIConfig) AIProvider {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "openai_compatible", "openai-compatible", "openai":
		return &OpenAICompatibleProvider{Config: cfg}
	default:
		return &MockAIProvider{Config: cfg}
	}
}

type MockAIProvider struct {
	Config models.AIConfig
}

func (p *MockAIProvider) Test() error { return nil }

func (p *MockAIProvider) Generate(prompt string, maxTokens int) (string, error) {
	if strings.Contains(prompt, "study planning agent") {
		return "", fmt.Errorf("mock provider does not generate structured plans")
	}
	return "今天的专注，会成为明天的底气。", nil
}

type OpenAICompatibleProvider struct {
	Config models.AIConfig
}

func (p *OpenAICompatibleProvider) Test() error {
	_, err := p.Generate("ping", 16)
	return err
}

func (p *OpenAICompatibleProvider) Generate(prompt string, maxTokens int) (string, error) {
	if strings.TrimSpace(p.Config.BaseURL) == "" {
		return "", fmt.Errorf("base url is required")
	}
	if strings.TrimSpace(p.Config.ModelName) == "" {
		return "", fmt.Errorf("model name is required")
	}
	if !p.Config.Enabled {
		return "", fmt.Errorf("provider is disabled")
	}
	reqBody := map[string]any{
		"model":       p.Config.ModelName,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0,
		"max_tokens":  maxInt(maxTokens, 64),
	}
	body, _ := json.Marshal(reqBody)
	client := &http.Client{Timeout: time.Duration(maxInt(p.Config.RequestTimeoutSeconds, 30)) * time.Second}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest(http.MethodPost, strings.TrimRight(p.Config.BaseURL, "/")+"/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		if key := strings.TrimSpace(decodeAIKey(p.Config)); key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("provider returned http %d", resp.StatusCode)
			continue
		}
		var decoded struct {
			Choices []struct {
				Message struct {
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(responseBody, &decoded); err != nil || len(decoded.Choices) == 0 {
			lastErr = fmt.Errorf("provider returned invalid completion")
			continue
		}
		content, err := decodeCompletionContent(decoded.Choices[0].Message.Content)
		if err != nil {
			lastErr = err
			continue
		}
		return content, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("provider request failed")
	}
	return "", lastErr
}

func decodeCompletionContent(raw json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if text = strings.TrimSpace(text); text != "" {
			return text, nil
		}
		return "", fmt.Errorf("provider returned empty completion")
	}
	if len(raw) == 0 || string(raw) == "null" || !json.Valid(raw) {
		return "", fmt.Errorf("provider returned invalid completion")
	}
	return string(raw), nil
}

func decodeAIKey(cfg models.AIConfig) string {
	if cfg.APIKeyCiphertext == "" {
		return ""
	}
	if !cfg.APIKeyEncrypted {
		return cfg.APIKeyCiphertext
	}
	return cfg.APIKeyCiphertext
}

func maxInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func CurrentAIProvider() (models.AIConfig, AIProvider, error) {
	cfg, err := loadAIConfig()
	if err != nil {
		return cfg, nil, err
	}
	return cfg, NewAIProvider(cfg), nil
}

func ProviderTestError(cfg models.AIConfig) error {
	if !cfg.Enabled {
		return fmt.Errorf("provider is disabled")
	}
	if strings.TrimSpace(cfg.Provider) == "" {
		cfg.Provider = "mock"
	}
	if strings.EqualFold(cfg.Provider, "openai_compatible") || strings.EqualFold(cfg.Provider, "openai-compatible") || strings.EqualFold(cfg.Provider, "openai") {
		if strings.TrimSpace(cfg.BaseURL) == "" {
			return fmt.Errorf("base url is required")
		}
		if strings.TrimSpace(cfg.ModelName) == "" {
			return fmt.Errorf("model name is required")
		}
	}
	return NewAIProvider(cfg).Test()
}

func GetAIProviderConfig() models.AIConfig {
	cfg, _ := loadAIConfig()
	return cfg
}

func SetAIProviderConfig(cfg models.AIConfig) error {
	// Keep as a thin hook for future planning-agent integration.
	return config.App.Validate()
}

func loadAIConfig() (models.AIConfig, error) {
	var cfg models.AIConfig
	err := db.DB.Order("id ASC").First(&cfg).Error
	if err != nil {
		cfg = models.AIConfig{Provider: "mock", RequestTimeoutSeconds: 30, DailyGenerationLimit: 5, Enabled: true}
		if createErr := db.DB.Create(&cfg).Error; createErr != nil {
			return cfg, createErr
		}
	}
	return cfg, err
}
