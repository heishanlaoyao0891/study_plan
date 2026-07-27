package services

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"study_plan_backend/db"
	"study_plan_backend/models"
)

var promptPatternGuidance = map[string]string{
	"truncated_output":  "确保在 token 上限内闭合全部 JSON 数组和对象，精简描述而不是省略尾部任务。",
	"invalid_json":      "仅返回可由标准 JSON 解析器读取的对象，不要注释、Markdown 或尾随逗号。",
	"order_reset":       "task.order 必须在所有阶段中全局连续递增，切换阶段时不得从 1 重置。",
	"capacity_overflow": "所有任务 effort_minutes 总和不得超过 capacity_minutes。",
	"invalid_schema":    "严格使用约定字段并补齐所有必填字段。",
}

func ClassifyBlueprintFailure(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "provider configuration"), strings.Contains(message, "provider is disabled"), strings.Contains(message, "deadline exceeded"), strings.Contains(message, "provider returned http"):
		return ""
	case IsAIOutputTruncated(err), strings.Contains(message, "truncated"):
		return "truncated_output"
	case strings.Contains(message, "invalid json"), strings.Contains(message, "unexpected end"):
		return "invalid_json"
	case strings.Contains(message, "effort") && strings.Contains(message, "capacity"):
		return "capacity_overflow"
	case strings.Contains(message, "order"):
		return "order_reset"
	default:
		return "invalid_schema"
	}
}

func RecordPromptPattern(ctx context.Context, key string) {
	if db.DB == nil || key == "" {
		return
	}
	guidance := promptPatternGuidance[key]
	if guidance == "" {
		return
	}
	_ = db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pattern models.AIPromptPattern
		err := tx.Where("pattern_key = ?", key).First(&pattern).Error
		if err == nil {
			return tx.Model(&pattern).UpdateColumn("count", gorm.Expr("count + 1")).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&models.AIPromptPattern{PatternKey: key, Version: 1, Count: 1, Guidance: guidance}).Error
	})
}

func ActivePromptPlaybookGuidance(ctx context.Context) string {
	if db.DB == nil {
		return ""
	}
	var patterns []models.AIPromptPattern
	if err := db.DB.WithContext(ctx).Where("count >= ?", 2).Order("count DESC").Limit(5).Find(&patterns).Error; err != nil {
		return ""
	}
	if len(patterns) == 0 {
		return ""
	}
	parts := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		parts = append(parts, pattern.Guidance)
	}
	return "\nPROMPT_PLAYBOOK_V1: " + strings.Join(parts, " ")
}
