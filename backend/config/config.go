package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv                 string // local/development/production
	Port                   string // 服务端口
	DBPath                 string // SQLite 数据库文件路径
	JWTSecret              string // JWT 签名密钥
	JWTExpireHours         int    // JWT 过期小时数
	WeChatAppID            string // 微信小程序 AppID
	WeChatSecret           string // 微信小程序 AppSecret
	WeChatLoginMock        bool   // 是否使用 mock 模式（无 AppID 时直接用 code 当作 openid）
	PhoneBindingRequired   bool   // 是否强制要求微信手机号验证；个人主体小程序无法使用该能力
	AdminUsername          string // PC 管理台初始管理员用户名
	AdminPassword          string // PC 管理台初始管理员密码
	AIKeySecret            string // AI API Key 服务端加密密钥
	AvatarStorage          string // local/cos/minio/object-url
	AvatarBaseURL          string // 头像对象存储公开 HTTPS base URL
	MakeupCostRatio        float64
	ArchiveEnabled         bool   // 是否启用 MySQL 归档同步
	ArchiveDriver          string // mysql
	ArchiveDSN             string // 归档库 DSN
	ArchiveIntervalMinutes int
	ArchiveTables          string
}

var App *Config

func Load() *Config {
	App = &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		Port:                   getEnv("PORT", "8080"),
		DBPath:                 getEnv("DB_PATH", "study_plan.db"),
		JWTSecret:              getEnv("JWT_SECRET", "study_plan_default_secret_change_me"),
		JWTExpireHours:         getEnvInt("JWT_EXPIRE_HOURS", 24*7),
		WeChatAppID:            getEnv("WECHAT_APPID", ""),
		WeChatSecret:           getEnv("WECHAT_SECRET", ""),
		WeChatLoginMock:        getEnvBool("WECHAT_LOGIN_MOCK", false),
		PhoneBindingRequired:   getEnvBool("PHONE_BINDING_REQUIRED", false),
		AdminUsername:          getEnv("ADMIN_USERNAME", ""),
		AdminPassword:          getEnv("ADMIN_PASSWORD", ""),
		AIKeySecret:            getEnv("AI_KEY_ENCRYPTION_SECRET", ""),
		AvatarStorage:          getEnv("AVATAR_STORAGE", "local"),
		AvatarBaseURL:          getEnv("AVATAR_BASE_URL", ""),
		MakeupCostRatio:        getEnvFloat("MAKEUP_COST_RATIO", 1),
		ArchiveEnabled:         getEnvBool("ARCHIVE_ENABLED", false),
		ArchiveDriver:          getEnv("ARCHIVE_DRIVER", "mysql"),
		ArchiveDSN:             getEnv("ARCHIVE_DSN", ""),
		ArchiveIntervalMinutes: getEnvInt("ARCHIVE_INTERVAL_MINUTES", 5),
		ArchiveTables:          getEnv("ARCHIVE_TABLES", "users,plans,daily_tasks,checkins,study_sessions,slack_records"),
	}
	return App
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func (c *Config) Validate() error {
	if !c.IsProduction() {
		return nil
	}
	missing := make([]string, 0)
	if c.JWTSecret == "" || c.JWTSecret == "study_plan_default_secret_change_me" || c.JWTSecret == "change-me-before-deploy" {
		missing = append(missing, "JWT_SECRET")
	}
	if c.DBPath == "" {
		missing = append(missing, "DB_PATH")
	}
	if c.WeChatAppID == "" {
		missing = append(missing, "WECHAT_APPID")
	}
	if c.WeChatSecret == "" {
		missing = append(missing, "WECHAT_SECRET")
	}
	if c.WeChatLoginMock {
		missing = append(missing, "WECHAT_LOGIN_MOCK=false")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid production configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return def
}
