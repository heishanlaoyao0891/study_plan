package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port            string // 服务端口
	DBPath          string // SQLite 数据库文件路径
	JWTSecret       string // JWT 签名密钥
	JWTExpireHours  int    // JWT 过期小时数
	WeChatAppID     string // 微信小程序 AppID
	WeChatSecret    string // 微信小程序 AppSecret
	WeChatLoginMock bool   // 是否使用 mock 模式（无 AppID 时直接用 code 当作 openid）
	AdminUsername   string // PC 管理台初始管理员用户名
	AdminPassword   string // PC 管理台初始管理员密码
}

var App *Config

func Load() *Config {
	App = &Config{
		Port:            getEnv("PORT", "8080"),
		DBPath:          getEnv("DB_PATH", "study_plan.db"),
		JWTSecret:       getEnv("JWT_SECRET", "study_plan_default_secret_change_me"),
		JWTExpireHours:  getEnvInt("JWT_EXPIRE_HOURS", 24*7),
		WeChatAppID:     getEnv("WECHAT_APPID", ""),
		WeChatSecret:    getEnv("WECHAT_SECRET", ""),
		WeChatLoginMock: getEnvBool("WECHAT_LOGIN_MOCK", false),
		AdminUsername:   getEnv("ADMIN_USERNAME", ""),
		AdminPassword:   getEnv("ADMIN_PASSWORD", ""),
	}
	if App.WeChatAppID == "" || App.WeChatSecret == "" || App.WeChatLoginMock {
		App.WeChatLoginMock = true
	}
	return App
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
