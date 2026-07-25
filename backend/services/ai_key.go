package services

import (
	"fmt"
	"strings"

	"study_plan_backend/aikey"
	"study_plan_backend/config"
	"study_plan_backend/models"
)

func ProtectAIAPIKey(secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", fmt.Errorf("api key is required")
	}
	if config.App == nil || strings.TrimSpace(config.App.AIKeySecret) == "" {
		return "", fmt.Errorf("AI_KEY_ENCRYPTION_SECRET must be configured before saving an API key")
	}
	return aikey.Encrypt(secret, config.App.AIKeySecret)
}

func DecodeAIAPIKey(cfg models.AIConfig) (string, error) {
	if cfg.APIKeyCiphertext == "" {
		return "", nil
	}
	if !cfg.APIKeyEncrypted {
		return cfg.APIKeyCiphertext, nil
	}
	if config.App == nil || strings.TrimSpace(config.App.AIKeySecret) == "" {
		return "", fmt.Errorf("AI_KEY_ENCRYPTION_SECRET is not configured")
	}
	return aikey.Decrypt(cfg.APIKeyCiphertext, config.App.AIKeySecret)
}
