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
