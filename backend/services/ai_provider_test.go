package services

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAIAPIKeyEncryptionRoundTrip(t *testing.T) {
	config.App = &config.Config{AIKeySecret: "test-only-encryption-secret"}
	encrypted, err := ProtectAIAPIKey("sk-test-value")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "sk-test-value" || strings.Contains(encrypted, "sk-test-value") {
		t.Fatal("API key was not encrypted")
	}
	decrypted, err := DecodeAIAPIKey(models.AIConfig{APIKeyCiphertext: encrypted, APIKeyEncrypted: true})
	if err != nil || decrypted != "sk-test-value" {
		t.Fatalf("encrypted key did not round trip: value=%q err=%v", decrypted, err)
	}
}

func TestProtectAIAPIKeyRejectsMissingEncryptionSecret(t *testing.T) {
	config.App = &config.Config{}
	if _, err := ProtectAIAPIKey("sk-test-value"); err == nil {
		t.Fatal("expected missing encryption secret to reject key storage")
	}
}

func usePublicTestServer(t *testing.T, server *httptest.Server) string {
	t.Helper()
	resetProviderClientCache()
	address := strings.TrimPrefix(server.URL, "http://")
	oldLookup, oldDial := lookupProviderIP, dialProvider
	lookupProviderIP = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
	}
	dialProvider = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	t.Cleanup(func() {
		resetProviderClientCache()
		lookupProviderIP, dialProvider = oldLookup, oldDial
	})
	return "http://provider.example"
}

func TestProviderClientCacheReusesTransportAndRefreshesCredentials(t *testing.T) {
	resetProviderClientCache()
	t.Cleanup(resetProviderClientCache)
	lookups := 0
	oldLookup := lookupProviderIP
	lookupProviderIP = func(context.Context, string) ([]net.IPAddr, error) {
		lookups++
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
	}
	t.Cleanup(func() { lookupProviderIP = oldLookup })
	cfg := models.AIConfig{BaseURL: "https://provider.example/v1", APIKeyCiphertext: "key-one"}
	first, err := reusableProviderClientContext(context.Background(), cfg, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := reusableProviderClientContext(context.Background(), cfg, time.Minute)
	if err != nil || first != second || lookups != 1 {
		t.Fatalf("provider client was not reused: same=%v lookups=%d err=%v", first == second, lookups, err)
	}
	cfg.APIKeyCiphertext = "key-two"
	refreshed, err := reusableProviderClientContext(context.Background(), cfg, time.Minute)
	if err != nil || refreshed == first || lookups != 2 {
		t.Fatalf("credential refresh did not replace cached client: replaced=%v lookups=%d err=%v", refreshed != first, lookups, err)
	}
}

func TestProviderDialRejectsDNSRebinding(t *testing.T) {
	resetProviderClientCache()
	t.Cleanup(resetProviderClientCache)
	lookups := 0
	oldLookup, oldDial := lookupProviderIP, dialProvider
	lookupProviderIP = func(context.Context, string) ([]net.IPAddr, error) {
		lookups++
		if lookups == 1 {
			return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
		}
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
	}
	dialProvider = func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("dial must not run after private DNS rebinding")
		return nil, nil
	}
	t.Cleanup(func() { lookupProviderIP, dialProvider = oldLookup, oldDial })
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "model", BaseURL: "http://provider.example/v1", RequestTimeoutSeconds: 5, Enabled: true, APIKeyCiphertext: "key"}
	_, err := NewAIProvider(cfg).Generate("test", 64)
	if err == nil || !strings.Contains(err.Error(), "local or private") || lookups < 2 {
		t.Fatalf("DNS rebinding was not blocked at dial time: lookups=%d err=%v", lookups, err)
	}
}

func TestProviderCapturesTokenUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{map[string]interface{}{"message": map[string]string{"content": `{"ok":true}`}}},
			"usage":   map[string]int{"prompt_tokens": 12, "completion_tokens": 7, "total_tokens": 19},
		})
	}))
	defer server.Close()
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "model", BaseURL: usePublicTestServer(t, server), RequestTimeoutSeconds: 5, Enabled: true, APIKeyCiphertext: "key"}
	telemetry := &AIProviderTelemetry{}
	if _, err := NewAIProvider(cfg).GenerateContext(WithAIProviderTelemetry(context.Background(), telemetry), "test", 64); err != nil {
		t.Fatal(err)
	}
	if telemetry.PromptTokens != 12 || telemetry.CompletionTokens != 7 || telemetry.TotalTokens != 19 {
		t.Fatalf("token usage was not captured: %+v", telemetry)
	}
}

func TestOpenAICompatibleProviderTestValidatesStructuredPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected completion path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("missing bearer key")
		}
		content := `{"title":"Connection test","summary":"Valid","estimated_total_hours":1,"rationale":"Test","tasks":[{"date":"2026-01-02","planned_start":"20:00","planned_end":"21:00","title":"Lesson","objective":"Complete one lesson exercise","description":"Test task","estimated_minutes":60,"difficulty":"easy"}]}`
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]string{"content": content}}}})
	}))
	defer server.Close()

	config.App = &config.Config{}
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "test-model", BaseURL: usePublicTestServer(t, server), RequestTimeoutSeconds: 5, DailyGenerationLimit: 5, Enabled: true, APIKeyCiphertext: "sk-test"}
	if err := ProviderTestError(cfg); err != nil {
		t.Fatalf("expected valid structured plan test to pass: %v", err)
	}
}

func TestSiliconFlowRequestsExplicitPlanJSON(t *testing.T) {
	request := buildCompletionRequest(models.AIConfig{Provider: AIProviderSiliconFlow, ModelName: SiliconFlowRecommendedModel}, "Create a plan", 768)
	responseFormat, _ := request["response_format"].(map[string]string)
	if responseFormat["type"] != "json_object" {
		t.Fatalf("SiliconFlow request did not require JSON output: %+v", responseFormat)
	}
	if thinking, ok := request["enable_thinking"].(bool); !ok || thinking {
		t.Fatalf("structured planning must disable thinking output: %+v", request["enable_thinking"])
	}
	messages, _ := request["messages"].([]map[string]string)
	if len(messages) < 2 || !strings.Contains(messages[0]["content"], `"tasks":[`) {
		t.Fatalf("prompt did not provide an exact tasks-array contract: %+v", messages)
	}
}

func TestProviderTestRejectsUnstructuredCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]string{"content": "pong"}}}})
	}))
	defer server.Close()

	config.App = &config.Config{}
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "test-model", BaseURL: usePublicTestServer(t, server) + "/v1", RequestTimeoutSeconds: 5, DailyGenerationLimit: 5, Enabled: true, APIKeyCiphertext: "sk-test"}
	err := ProviderTestError(cfg)
	if err == nil || !strings.Contains(err.Error(), "structured plan") {
		t.Fatalf("expected structured plan validation error, got %v", err)
	}
}

func TestValidateAIConfigRejectsUnsafeOrUnknownConfiguration(t *testing.T) {
	oldLookup := lookupProviderIP
	lookupProviderIP = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
	}
	t.Cleanup(func() { lookupProviderIP = oldLookup })
	config.App = &config.Config{AppEnv: "production"}
	base := models.AIConfig{Provider: AIProviderSiliconFlow, ModelName: SiliconFlowRecommendedModel, BaseURL: SiliconFlowBaseURL, RequestTimeoutSeconds: 30, DailyGenerationLimit: 5, Enabled: true, APIKeyCiphertext: "encrypted", APIKeyEncrypted: true}
	if err := ValidateAIConfig(base, true); err != nil {
		t.Fatalf("expected recommended SiliconFlow config to pass: %v", err)
	}
	base.BaseURL = "http://api.siliconflow.cn/v1"
	if err := ValidateAIConfig(base, true); err == nil {
		t.Fatal("expected production HTTP base URL to fail")
	}
	base.BaseURL = "https://api.siliconflow.cn"
	if err := ValidateAIConfig(base, true); err == nil {
		t.Fatal("expected SiliconFlow non-canonical path to fail")
	}
	base.Provider = "mystery"
	if err := ValidateAIConfig(base, true); err == nil {
		t.Fatal("expected unsupported provider to fail")
	}
}

func TestValidateAIConfigRejectsPrivateLiteralAndResolvedDestinations(t *testing.T) {
	config.App = &config.Config{}
	base := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "model", RequestTimeoutSeconds: 30, DailyGenerationLimit: 5, Enabled: true, APIKeyCiphertext: "key"}
	for _, baseURL := range []string{"http://127.0.0.1", "http://[::1]", "http://169.254.169.254", "http://10.0.0.1", "http://localhost"} {
		base.BaseURL = baseURL
		if err := ValidateAIConfig(base, false); err == nil {
			t.Errorf("expected %s to be rejected", baseURL)
		}
	}
	oldLookup := lookupProviderIP
	lookupProviderIP = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("192.168.1.10")}}, nil
	}
	t.Cleanup(func() { lookupProviderIP = oldLookup })
	base.BaseURL = "https://provider.example"
	if err := ValidateAIConfig(base, false); err == nil {
		t.Fatal("expected a hostname resolving privately to be rejected")
	}
}

func TestProviderResponseBodyIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", maxProviderResponseBytes+1)))
	}))
	defer server.Close()
	config.App = &config.Config{}
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "model", BaseURL: usePublicTestServer(t, server), RequestTimeoutSeconds: 5, DailyGenerationLimit: 5, Enabled: true, APIKeyCiphertext: "key"}
	_, err := NewAIProvider(cfg).Generate("test", 64)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected bounded response error, got %v", err)
	}
}

func TestProviderDoesNotRetryInvalidOutput(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()
	config.App = &config.Config{}
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "model", BaseURL: usePublicTestServer(t, server), RequestTimeoutSeconds: 5, DailyGenerationLimit: 5, Enabled: true, APIKeyCiphertext: "key"}
	if _, err := NewAIProvider(cfg).Generate("test", 64); err == nil {
		t.Fatal("expected invalid output error")
	}
	if attempts != 1 {
		t.Fatalf("invalid output must not be retried, got %d attempts", attempts)
	}
}

func TestProvider429RetriesConsumeQuotaPerAttempt(t *testing.T) {
	setupAIUsageTestDB(t)
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	config.App = &config.Config{}
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "model", BaseURL: usePublicTestServer(t, server), RequestTimeoutSeconds: 5, DailyGenerationLimit: 2, Enabled: true, APIKeyCiphertext: "key"}
	ctx := WithAIQuota(context.Background(), 4, cfg.Provider, 2, nil)
	if _, err := NewAIProvider(cfg).GenerateContext(ctx, "test", 64); err == nil {
		t.Fatal("expected provider 429 failure")
	}
	_, count, err := CanUseAIGeneration(4, 2)
	if err != nil || attempts != 2 || count != 2 {
		t.Fatalf("each 429 attempt must consume quota: attempts=%d count=%d err=%v", attempts, count, err)
	}
}

func TestProviderRedirectCannotChangeOrigin(t *testing.T) {
	finalReached := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			http.Redirect(w, r, "http://sub.provider.example/final", http.StatusFound)
			return
		}
		finalReached = true
	}))
	defer server.Close()
	config.App = &config.Config{}
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, ModelName: "model", BaseURL: usePublicTestServer(t, server), RequestTimeoutSeconds: 5, DailyGenerationLimit: 5, Enabled: true, APIKeyCiphertext: "stored-key"}
	_, err := NewAIProvider(cfg).Generate("test", 64)
	if err == nil || !strings.Contains(err.Error(), "changed origin") || finalReached {
		t.Fatalf("expected cross-origin redirect rejection before sending credentials: reached=%v err=%v", finalReached, err)
	}
}

func TestLoadAIConfigNormalizesLegacyProviderAlias(t *testing.T) {
	connection, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	db.DB = connection
	if err := db.DB.AutoMigrate(&models.AIConfig{}); err != nil {
		t.Fatal(err)
	}
	legacy := models.AIConfig{Provider: "openai-compatible"}
	if err := db.DB.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	loaded, err := loadAIConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Provider != AIProviderOpenAICompatible {
		t.Fatalf("expected normalized provider, got %q", loaded.Provider)
	}
	var persisted models.AIConfig
	if err := db.DB.First(&persisted, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Provider != AIProviderOpenAICompatible {
		t.Fatalf("expected normalized provider persisted, got %q", persisted.Provider)
	}
}

func TestNormalizeAIProviderAliases(t *testing.T) {
	for _, alias := range []string{"openai", "openai-compatible", " OPENAI_COMPATIBLE "} {
		if got := NormalizeAIProvider(alias); got != AIProviderOpenAICompatible {
			t.Errorf("NormalizeAIProvider(%q) = %q", alias, got)
		}
	}
	if got := NormalizeAIProvider(" deepseek "); got != AIProviderSiliconFlow {
		t.Errorf("expected legacy deepseek normalized to siliconflow, got %q", got)
	}
}

func TestNormalizeLegacyDeepSeekConfigPreservesCustomValues(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		modelName string
		wantURL   string
		wantModel string
	}{
		{name: "old defaults", baseURL: legacyDeepSeekBaseURL, modelName: legacyDeepSeekModel, wantURL: SiliconFlowBaseURL, wantModel: SiliconFlowRecommendedModel},
		{name: "empty URL", baseURL: "", modelName: legacyDeepSeekModel, wantURL: SiliconFlowBaseURL, wantModel: SiliconFlowRecommendedModel},
		{name: "custom URL", baseURL: "https://gateway.example/v1", modelName: legacyDeepSeekModel, wantURL: "https://gateway.example/v1", wantModel: legacyDeepSeekModel},
		{name: "custom model", baseURL: legacyDeepSeekBaseURL, modelName: "custom-model", wantURL: SiliconFlowBaseURL, wantModel: "custom-model"},
		{name: "custom values", baseURL: "https://gateway.example/v1", modelName: "custom-model", wantURL: "https://gateway.example/v1", wantModel: "custom-model"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := models.AIConfig{Provider: "deepseek", BaseURL: tt.baseURL, ModelName: tt.modelName}
			NormalizeAIConfig(&cfg)
			wantProvider := AIProviderSiliconFlow
			if tt.baseURL == "https://gateway.example/v1" {
				wantProvider = AIProviderOpenAICompatible
			}
			if cfg.Provider != wantProvider || cfg.BaseURL != tt.wantURL || cfg.ModelName != tt.wantModel {
				t.Fatalf("unexpected normalized config: %+v", cfg)
			}
		})
	}
}

func TestCompletionEndpointHandlesVersionedBaseURL(t *testing.T) {
	if got := completionEndpoint(SiliconFlowBaseURL); got != "https://api.siliconflow.cn/v1/chat/completions" {
		t.Fatalf("unexpected SiliconFlow completion endpoint %q", got)
	}
	if got := completionEndpoint("https://provider.example"); got != "https://provider.example/v1/chat/completions" {
		t.Fatalf("unexpected unversioned completion endpoint %q", got)
	}
}

func TestDisabledProviderDoesNotGenerate(t *testing.T) {
	cfg := models.AIConfig{Provider: AIProviderOpenAICompatible, Enabled: false}
	_, err := NewAIProvider(cfg).Generate("test", 64)
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected disabled provider rejection, got %v", err)
	}
}
