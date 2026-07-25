package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"study_plan_backend/config"
)

type WeChatSession struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg,omitempty"`
}

type weChatAccessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg,omitempty"`
}

type weChatSubscriptionSendReq struct {
	ToUser           string         `json:"touser"`
	TemplateID       string         `json:"template_id"`
	Page             string         `json:"page,omitempty"`
	Data             map[string]any `json:"data"`
	MiniProgramState string         `json:"miniprogram_state,omitempty"`
}

const weChatLoginURL = "https://api.weixin.qq.com/sns/jscode2session"
const weChatAccessTokenURL = "https://api.weixin.qq.com/cgi-bin/token"

var weChatHTTPClient = http.DefaultClient
var tokenCache struct {
	sync.Mutex
	value     string
	expiresAt time.Time
}

// Code2Session 用小程序 code 换取 openid
// 仅当 WECHAT_LOGIN_MOCK=true 时走 mock 模式，生产环境必须调用微信 code2session。
func Code2Session(code string) (*WeChatSession, error) {
	cfg := config.App

	if cfg.WeChatLoginMock {
		// mock：用 code 派生一个稳定的 openid
		return &WeChatSession{
			OpenID:     fmt.Sprintf("mock_%s", code),
			SessionKey: "mock_session_key",
		}, nil
	}
	if cfg.WeChatAppID == "" || cfg.WeChatSecret == "" {
		return nil, errors.New("wechat appid or secret is not configured")
	}

	url := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		weChatLoginURL, cfg.WeChatAppID, cfg.WeChatSecret, code)

	resp, err := weChatHTTPClient.Get(url)
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
		return nil, fmt.Errorf("wechat code2session failed: errcode=%d errmsg=%s", s.ErrCode, s.ErrMsg)
	}
	return &s, nil
}

func getWeChatAccessToken() (string, error) {
	tokenCache.Lock()
	defer tokenCache.Unlock()
	if tokenCache.value != "" && time.Now().Before(tokenCache.expiresAt) {
		return tokenCache.value, nil
	}
	cfg := config.App
	endpoint := fmt.Sprintf("%s?grant_type=client_credential&appid=%s&secret=%s", weChatAccessTokenURL, url.QueryEscape(cfg.WeChatAppID), url.QueryEscape(cfg.WeChatSecret))
	resp, err := weChatHTTPClient.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("call wechat access token api: %w", err)
	}
	defer resp.Body.Close()
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read wechat access token body: %w", err)
	}
	var result weChatAccessTokenResp
	if err := json.Unmarshal(responseData, &result); err != nil {
		return "", fmt.Errorf("parse wechat access token response: %w", err)
	}
	if result.ErrCode != 0 || result.AccessToken == "" {
		return "", fmt.Errorf("wechat access token failed: errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}
	expiresIn := result.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	tokenCache.value = result.AccessToken
	tokenCache.expiresAt = time.Now().Add(time.Duration(expiresIn)*time.Second - 5*time.Minute)
	return result.AccessToken, nil
}

func SendSubscriptionMessage(openid, templateID, page string, data map[string]any) error {
	cfg := config.App
	if cfg.WeChatLoginMock {
		return nil
	}
	token, err := getWeChatAccessToken()
	if err != nil {
		return err
	}
	payload := weChatSubscriptionSendReq{
		ToUser:           openid,
		TemplateID:       templateID,
		Page:             page,
		Data:             data,
		MiniProgramState: "formal",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s", token), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := weChatHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(responseData, &result); err != nil {
		return err
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wechat subscription send failed: errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}
	return nil
}
