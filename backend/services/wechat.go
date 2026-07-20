package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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

type weChatPhoneResp struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg,omitempty"`
	PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
	} `json:"phone_info"`
}

const weChatLoginURL = "https://api.weixin.qq.com/sns/jscode2session"
const weChatAccessTokenURL = "https://api.weixin.qq.com/cgi-bin/token"
const weChatPhoneURL = "https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s"

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
		return nil, fmt.Errorf("wechat code2session failed: errcode=%d errmsg=%s", s.ErrCode, s.ErrMsg)
	}
	return &s, nil
}

func GetPhoneNumber(code string) (string, error) {
	cfg := config.App
	if cfg.WeChatLoginMock {
		phone := strings.TrimSpace(code)
		if phone == "" {
			phone = "13800000000"
		}
		return phone, nil
	}
	if cfg.WeChatAppID == "" || cfg.WeChatSecret == "" {
		return "", errors.New("wechat appid or secret is not configured")
	}
	token, err := getWeChatAccessToken()
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]string{"code": code})
	resp, err := http.Post(fmt.Sprintf(weChatPhoneURL, token), "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("call wechat phone api: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read wechat phone body: %w", err)
	}
	var result weChatPhoneResp
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parse wechat phone response: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat phone api failed: errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}
	phone := result.PhoneInfo.PurePhoneNumber
	if phone == "" {
		phone = result.PhoneInfo.PhoneNumber
	}
	if phone == "" {
		return "", errors.New("wechat phone api returned empty phone number")
	}
	return phone, nil
}

func getWeChatAccessToken() (string, error) {
	cfg := config.App
	url := fmt.Sprintf("%s?grant_type=client_credential&appid=%s&secret=%s", weChatAccessTokenURL, cfg.WeChatAppID, cfg.WeChatSecret)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("call wechat access token api: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read wechat access token body: %w", err)
	}
	var result weChatAccessTokenResp
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parse wechat access token response: %w", err)
	}
	if result.ErrCode != 0 || result.AccessToken == "" {
		return "", fmt.Errorf("wechat access token failed: errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}
	return result.AccessToken, nil
}
