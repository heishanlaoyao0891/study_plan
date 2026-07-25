package services

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"study_plan_backend/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestAccessTokenIsCached(t *testing.T) {
	oldClient := weChatHTTPClient
	oldConfig := config.App
	t.Cleanup(func() {
		weChatHTTPClient = oldClient
		config.App = oldConfig
		tokenCache.Lock()
		tokenCache.value, tokenCache.expiresAt = "", time.Time{}
		tokenCache.Unlock()
	})
	config.App = &config.Config{WeChatAppID: "app", WeChatSecret: "secret"}
	tokenCache.value, tokenCache.expiresAt = "", time.Time{}
	calls := 0
	weChatHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"cached-token","expires_in":7200}`)), Header: make(http.Header)}, nil
	})}

	for i := 0; i < 2; i++ {
		token, err := getWeChatAccessToken()
		if err != nil || token != "cached-token" {
			t.Fatalf("token=%q err=%v", token, err)
		}
	}
	if calls != 1 {
		t.Fatalf("expected one token request, got %d", calls)
	}
}
