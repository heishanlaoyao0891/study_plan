package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"study_plan_backend/config"
)

type WeChatSession struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg,omitempty"`
}

const weChatLoginURL = "https://api.weixin.qq.com/sns/jscode2session"

// Code2Session 用小程序 code 换取 openid
// 当未配置 AppID 时走 mock 模式：直接把 code 当作 openid 返回（方便本地开发调试）
func Code2Session(code string) (*WeChatSession, error) {
	cfg := config.App

	if cfg.WeChatLoginMock {
		// mock：用 code 派生一个稳定的 openid
		return &WeChatSession{
			OpenID:     fmt.Sprintf("mock_%s", code),
			SessionKey: "mock_session_key",
		}, nil
	}

	url := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		weChatLoginURL, cfg.WeChatAppID, cfg.WeChatSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("call wechat: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wechat body: %w", err)
	}

	var s WeChatSession
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("parse wechat response: %w", err)
	}
	if s.ErrCode != 0 {
		return nil, errors.New(s.ErrMsg)
	}
	return &s, nil
}