package config

import "testing"

func TestLoadDefaultsToWechatMockOutsideProductionWhenCredentialsMissing(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")

	cfg := Load()
	if !cfg.WeChatLoginMock {
		t.Fatal("expected mock login when non-production WeChat credentials are missing")
	}
}

func TestLoadRespectsExplicitWechatMockSetting(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("WECHAT_APPID", "")
	t.Setenv("WECHAT_SECRET", "")
	t.Setenv("WECHAT_LOGIN_MOCK", "false")

	cfg := Load()
	if cfg.WeChatLoginMock {
		t.Fatal("expected explicit WECHAT_LOGIN_MOCK=false to be respected")
	}
}

func TestStagingEnvironmentIsNotProduction(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	cfg := Load()
	if cfg.IsProduction() {
		t.Fatal("staging must not be treated as production")
	}
	if !cfg.IsStaging() {
		t.Fatal("expected staging environment to be recognized")
	}
}

func TestProductionRequiresAIKeyEncryptionSecret(t *testing.T) {
	cfg := &Config{
		AppEnv:          "production",
		DBPath:          "study.db",
		JWTSecret:       "strong-jwt-secret",
		WeChatAppID:     "appid",
		WeChatSecret:    "wechat-secret",
		WeChatLoginMock: false,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production validation to require AI key encryption secret")
	}
	cfg.AIKeySecret = "strong-ai-key-secret"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected complete production config to pass: %v", err)
	}
}
