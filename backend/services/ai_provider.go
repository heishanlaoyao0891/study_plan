package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/models"
)

type AIProvider interface {
	Test() error
	Generate(prompt string, maxTokens int) (string, error)
	GenerateContext(ctx context.Context, prompt string, maxTokens int) (string, error)
}

type AIProviderTelemetry struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type aiProviderTelemetryKey struct{}

func WithAIProviderTelemetry(ctx context.Context, telemetry *AIProviderTelemetry) context.Context {
	return context.WithValue(ctx, aiProviderTelemetryKey{}, telemetry)
}

const (
	AIProviderMock              = "mock"
	AIProviderOpenAICompatible  = "openai_compatible"
	AIProviderSiliconFlow       = "siliconflow"
	SiliconFlowBaseURL          = "https://api.siliconflow.cn/v1"
	SiliconFlowRecommendedModel = "deepseek-ai/DeepSeek-V3.2"
	legacyDeepSeekBaseURL       = "https://api.deepseek.com"
	legacyDeepSeekModel         = "deepseek-chat"
	maxProviderResponseBytes    = 1 << 20
)

var (
	lookupProviderIP = net.DefaultResolver.LookupIPAddr
	dialProvider     = (&net.Dialer{}).DialContext
)

type cachedProviderClient struct {
	origin string
	client *http.Client
}

var providerClientCache = struct {
	sync.Mutex
	items map[string]cachedProviderClient
}{items: map[string]cachedProviderClient{}}

func NormalizeAIProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai", "openai-compatible", AIProviderOpenAICompatible:
		return AIProviderOpenAICompatible
	case "deepseek":
		return AIProviderSiliconFlow
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func NormalizeAIConfig(cfg *models.AIConfig) {
	if cfg.InteractiveTargetSeconds == 0 {
		cfg.InteractiveTargetSeconds = 2
	}
	if cfg.BackgroundJobTimeoutSeconds == 0 {
		cfg.BackgroundJobTimeoutSeconds = 60
	}
	legacyDeepSeek := strings.EqualFold(strings.TrimSpace(cfg.Provider), "deepseek")
	if !legacyDeepSeek {
		cfg.Provider = NormalizeAIProvider(cfg.Provider)
		return
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" || baseURL == legacyDeepSeekBaseURL {
		cfg.Provider = AIProviderSiliconFlow
		cfg.BaseURL = SiliconFlowBaseURL
		if strings.TrimSpace(cfg.ModelName) == legacyDeepSeekModel {
			cfg.ModelName = SiliconFlowRecommendedModel
		}
		return
	}
	cfg.Provider = AIProviderOpenAICompatible
}

func NewAIProvider(cfg models.AIConfig) AIProvider {
	switch NormalizeAIProvider(cfg.Provider) {
	case AIProviderOpenAICompatible, AIProviderSiliconFlow:
		return &OpenAICompatibleProvider{Config: cfg}
	case AIProviderMock:
		return &MockAIProvider{Config: cfg}
	default:
		return &UnsupportedAIProvider{Provider: cfg.Provider}
	}
}

type UnsupportedAIProvider struct{ Provider string }

func (p *UnsupportedAIProvider) Test() error {
	return fmt.Errorf("unsupported provider %q", p.Provider)
}
func (p *UnsupportedAIProvider) Generate(string, int) (string, error) {
	return "", p.Test()
}
func (p *UnsupportedAIProvider) GenerateContext(context.Context, string, int) (string, error) {
	return "", p.Test()
}

type MockAIProvider struct {
	Config models.AIConfig
}

func (p *MockAIProvider) Test() error { return nil }

func (p *MockAIProvider) Generate(prompt string, maxTokens int) (string, error) {
	return p.GenerateContext(context.Background(), prompt, maxTokens)
}

func (p *MockAIProvider) GenerateContext(_ context.Context, prompt string, maxTokens int) (string, error) {
	if strings.Contains(prompt, "study planning agent") {
		return "", fmt.Errorf("mock provider does not generate structured plans")
	}
	return "今天的专注，会成为明天的底气。", nil
}

type OpenAICompatibleProvider struct {
	Config models.AIConfig
}

func (p *OpenAICompatibleProvider) Test() error {
	prompt := "Create exactly one connection-test task dated 2026-01-02 from 20:00 to 21:00."
	raw, err := p.Generate(prompt, 768)
	if err != nil {
		return err
	}
	preview, err := ParsePlanPreviewJSON(raw)
	if err != nil {
		return fmt.Errorf("provider did not return structured plan JSON: %w", err)
	}
	if err := ValidatePlanPreview(preview, PlanGenerationInput{Goal: "connection test", Days: 1}); err != nil {
		return fmt.Errorf("provider returned invalid structured plan: %w", err)
	}
	return nil
}

func (p *OpenAICompatibleProvider) Generate(prompt string, maxTokens int) (string, error) {
	return p.GenerateContext(context.Background(), prompt, maxTokens)
}

func (p *OpenAICompatibleProvider) GenerateContext(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if !p.Config.Enabled {
		return "", fmt.Errorf("provider is disabled")
	}
	reqBody := buildCompletionRequest(p.Config, prompt, maxTokens)
	body, _ := json.Marshal(reqBody)
	timeout := time.Duration(maxInt(p.Config.RequestTimeoutSeconds, 30)) * time.Second
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < timeout {
		timeout = time.Until(deadline)
	}
	client, err := reusableProviderClientContext(ctx, p.Config, timeout)
	if err != nil {
		return "", err
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, completionEndpoint(p.Config.BaseURL), bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		key, err := DecodeAIAPIKey(p.Config)
		if err != nil {
			return "", err
		}
		if key = strings.TrimSpace(key); key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		if err := ReserveAIProviderAttempt(ctx); err != nil {
			return "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			continue
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxProviderResponseBytes+1))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if len(responseBody) > maxProviderResponseBytes {
			return "", fmt.Errorf("provider response exceeds %d bytes", maxProviderResponseBytes)
		}
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("provider returned http %d", resp.StatusCode)
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				return "", lastErr
			}
			continue
		}
		var decoded struct {
			Choices []struct {
				Message struct {
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(responseBody, &decoded); err != nil || len(decoded.Choices) == 0 {
			return "", fmt.Errorf("provider returned invalid completion")
		}
		content, err := decodeCompletionContent(decoded.Choices[0].Message.Content)
		if err != nil {
			return "", err
		}
		if telemetry, ok := ctx.Value(aiProviderTelemetryKey{}).(*AIProviderTelemetry); ok && telemetry != nil {
			telemetry.PromptTokens = decoded.Usage.PromptTokens
			telemetry.CompletionTokens = decoded.Usage.CompletionTokens
			telemetry.TotalTokens = decoded.Usage.TotalTokens
		}
		return content, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("provider request failed")
	}
	return "", lastErr
}

func buildCompletionRequest(cfg models.AIConfig, prompt string, maxTokens int) map[string]any {
	contract := `Return exactly one JSON object with this shape and these exact key names. Do not rename tasks to task, steps, schedule, or daily_tasks. Do not wrap the JSON in Markdown:
{"title":"string","summary":"string","estimated_total_hours":1,"rationale":"string","tasks":[{"date":"YYYY-MM-DD","planned_start":"HH:mm","planned_end":"HH:mm","title":"string","objective":"specific action different from title","description":"string","estimated_minutes":60,"difficulty":"easy"}]}`
	if strings.Contains(prompt, `"contract":"planning_blueprint_v1"`) {
		contract = `Return exactly one JSON object and no Markdown. Do not include dates, time slots, persisted IDs, or private data. Use this schema:
{"title":"string","summary":"string","rationale":"string","stages":[{"id":"stage_1","name":"string","objective":"string","order":1}],"tasks":[{"id":"task_1","stage_id":"stage_1","title":"string","objective":"specific action different from title","description":"string","effort_minutes":60,"difficulty":"easy|medium|hard","order":1,"prerequisite_ids":[]}]}`
	}
	request := map[string]any{
		"model":       cfg.ModelName,
		"messages":    []map[string]string{{"role": "system", "content": contract}, {"role": "user", "content": prompt}},
		"temperature": 0,
		"max_tokens":  maxInt(maxTokens, 64),
	}
	if NormalizeAIProvider(cfg.Provider) == AIProviderSiliconFlow {
		request["response_format"] = map[string]string{"type": "json_object"}
		request["enable_thinking"] = false
	}
	return request
}

func decodeCompletionContent(raw json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if text = strings.TrimSpace(text); text != "" {
			return text, nil
		}
		return "", fmt.Errorf("provider returned empty completion")
	}
	if len(raw) == 0 || string(raw) == "null" || !json.Valid(raw) {
		return "", fmt.Errorf("provider returned invalid completion")
	}
	return string(raw), nil
}

func maxInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func completionEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(strings.ToLower(baseURL), "/v1") {
		return baseURL + "/chat/completions"
	}
	return baseURL + "/v1/chat/completions"
}

func reusableProviderClientContext(ctx context.Context, cfg models.AIConfig, timeout time.Duration) (*http.Client, error) {
	key, origin, err := providerClientCacheKey(cfg)
	if err != nil {
		return nil, err
	}
	providerClientCache.Lock()
	if cached, ok := providerClientCache.items[key]; ok {
		providerClientCache.Unlock()
		return cached.client, nil
	}
	providerClientCache.Unlock()
	client, err := restrictedProviderClientContext(ctx, cfg.BaseURL, timeout)
	if err != nil {
		return nil, err
	}
	client.Timeout = 0
	providerClientCache.Lock()
	if cached, ok := providerClientCache.items[key]; ok {
		providerClientCache.Unlock()
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
		return cached.client, nil
	}
	for existingKey, cached := range providerClientCache.items {
		if cached.origin == origin && existingKey != key {
			if transport, ok := cached.client.Transport.(*http.Transport); ok {
				transport.CloseIdleConnections()
			}
			delete(providerClientCache.items, existingKey)
		}
	}
	providerClientCache.items[key] = cachedProviderClient{origin: origin, client: client}
	providerClientCache.Unlock()
	return client, nil
}

func providerClientCacheKey(cfg models.AIConfig) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("base_url must be an absolute HTTP(S) URL")
	}
	origin := strings.ToLower(parsed.Scheme + "://" + parsed.Host)
	payload := origin + "\x00" + cfg.APIKeyCiphertext + "\x00" + fmt.Sprintf("%t", cfg.APIKeyEncrypted)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), origin, nil
}

func resetProviderClientCache() {
	providerClientCache.Lock()
	defer providerClientCache.Unlock()
	for _, cached := range providerClientCache.items {
		if transport, ok := cached.client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	providerClientCache.items = map[string]cachedProviderClient{}
}

func restrictedProviderClient(baseURL string, timeout time.Duration) (*http.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return restrictedProviderClientContext(ctx, baseURL, timeout)
}

func restrictedProviderClientContext(ctx context.Context, baseURL string, timeout time.Duration) (*http.Client, error) {
	parsed, err := validateAIBaseURLContext(ctx, baseURL)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid provider address: %w", err)
		}
		addresses, err := resolvePublicProviderIPs(ctx, host)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, address := range addresses {
			conn, err := dialProvider(ctx, network, net.JoinHostPort(address.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
	client := &http.Client{Transport: transport, Timeout: timeout}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if _, err := validateAIBaseURL(req.URL.String()); err != nil {
			return err
		}
		if !SameAIProviderOrigin(parsed.String(), req.URL.String()) {
			return fmt.Errorf("provider redirect changed origin")
		}
		return nil
	}
	return client, nil
}

func validateAIBaseURL(baseURL string) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return validateAIBaseURLContext(ctx, baseURL)
}

func validateAIBaseURLContext(ctx context.Context, baseURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("base_url must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	if NormalizeAIProviderURLHost(parsed.Hostname()) == "" {
		return nil, fmt.Errorf("base_url host is invalid")
	}
	if _, err := resolvePublicProviderIPs(ctx, parsed.Hostname()); err != nil {
		return nil, err
	}
	return parsed, nil
}

func NormalizeAIProviderURLHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func SameAIProviderOrigin(first, second string) bool {
	firstURL, firstErr := url.Parse(strings.TrimSpace(first))
	secondURL, secondErr := url.Parse(strings.TrimSpace(second))
	if firstErr != nil || secondErr != nil || firstURL.Scheme == "" || secondURL.Scheme == "" {
		return false
	}
	return strings.EqualFold(firstURL.Scheme, secondURL.Scheme) && strings.EqualFold(firstURL.Host, secondURL.Host)
}

func resolvePublicProviderIPs(ctx context.Context, host string) ([]net.IPAddr, error) {
	host = NormalizeAIProviderURLHost(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, fmt.Errorf("base_url must not resolve to a local or private address")
	}
	var addresses []net.IPAddr
	if literal := net.ParseIP(host); literal != nil {
		addresses = []net.IPAddr{{IP: literal}}
	} else {
		var err error
		addresses, err = lookupProviderIP(ctx, host)
		if err != nil || len(addresses) == 0 {
			return nil, fmt.Errorf("base_url host could not be resolved")
		}
	}
	for _, address := range addresses {
		if !isPublicProviderIP(address.IP) {
			return nil, fmt.Errorf("base_url must not resolve to a local or private address")
		}
	}
	return addresses, nil
}

func isPublicProviderIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
}

func CurrentAIProvider() (models.AIConfig, AIProvider, error) {
	return CurrentAIProviderContext(context.Background())
}

func CurrentAIProviderContext(ctx context.Context) (models.AIConfig, AIProvider, error) {
	cfg, err := loadAIConfigContext(ctx)
	if err != nil {
		return cfg, nil, err
	}
	return cfg, NewAIProvider(cfg), nil
}

func ProviderTestError(cfg models.AIConfig) error {
	if !cfg.Enabled {
		return fmt.Errorf("provider is disabled")
	}
	if err := ValidateAIConfig(cfg, false); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(cfg.Provider), AIProviderMock) {
		return fmt.Errorf("mock mode uses deterministic fallback and cannot validate an AI connection")
	}
	return NewAIProvider(cfg).Test()
}

func ValidateAIConfig(cfg models.AIConfig, requireEncryptedKey bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ValidateAIConfigContext(ctx, cfg, requireEncryptedKey)
}

func ValidateAIConfigContext(ctx context.Context, cfg models.AIConfig, requireEncryptedKey bool) error {
	provider := NormalizeAIProvider(cfg.Provider)
	if provider != AIProviderMock && provider != AIProviderOpenAICompatible && provider != AIProviderSiliconFlow {
		return fmt.Errorf("provider must be one of mock, openai_compatible, or siliconflow")
	}
	if cfg.RequestTimeoutSeconds < 1 || cfg.RequestTimeoutSeconds > 120 {
		return fmt.Errorf("request_timeout_seconds must be between 1 and 120")
	}
	interactiveTarget := cfg.InteractiveTargetSeconds
	if interactiveTarget == 0 {
		interactiveTarget = 2
	}
	if interactiveTarget < 1 || interactiveTarget > 5 {
		return fmt.Errorf("interactive_target_seconds must be between 1 and 5")
	}
	backgroundBudget := cfg.BackgroundJobTimeoutSeconds
	if backgroundBudget == 0 {
		backgroundBudget = 60
	}
	if backgroundBudget < 15 || backgroundBudget > 120 {
		return fmt.Errorf("background_job_timeout_seconds must be between 15 and 120")
	}
	if cfg.DailyGenerationLimit < 1 || cfg.DailyGenerationLimit > 100 {
		return fmt.Errorf("daily_generation_limit must be between 1 and 100")
	}
	if !cfg.Enabled {
		return nil
	}
	if provider == AIProviderMock {
		return nil
	}
	if strings.TrimSpace(cfg.ModelName) == "" {
		return fmt.Errorf("model name is required")
	}
	parsed, err := validateAIBaseURLContext(ctx, cfg.BaseURL)
	if err != nil {
		return err
	}
	if provider == AIProviderSiliconFlow && (parsed.Scheme != "https" || NormalizeAIProviderURLHost(parsed.Hostname()) != "api.siliconflow.cn" || parsed.Port() != "" || parsed.EscapedPath() != "/v1") {
		return fmt.Errorf("siliconflow base_url must be exactly %s", SiliconFlowBaseURL)
	}
	if config.App != nil && config.App.IsProduction() && parsed.Scheme != "https" {
		return fmt.Errorf("base_url must use HTTPS in production")
	}
	if cfg.APIKeyCiphertext == "" {
		return fmt.Errorf("api key is required for %s", provider)
	}
	if requireEncryptedKey && !cfg.APIKeyEncrypted {
		return fmt.Errorf("API key must be encrypted at rest; configure AI_KEY_ENCRYPTION_SECRET and update the key")
	}
	return nil
}

func GetAIProviderConfig() models.AIConfig {
	cfg, _ := loadAIConfig()
	return cfg
}

func SetAIProviderConfig(cfg models.AIConfig) error {
	// Keep as a thin hook for future planning-agent integration.
	return config.App.Validate()
}

func loadAIConfig() (models.AIConfig, error) {
	return loadAIConfigContext(context.Background())
}

func loadAIConfigContext(ctx context.Context) (models.AIConfig, error) {
	var cfg models.AIConfig
	database := db.DB.WithContext(ctx)
	err := database.Order("id ASC").First(&cfg).Error
	if err != nil {
		cfg = models.AIConfig{Provider: AIProviderMock, RequestTimeoutSeconds: 30, InteractiveTargetSeconds: 2, BackgroundJobTimeoutSeconds: 60, DailyGenerationLimit: 5, Enabled: true}
		if createErr := database.Create(&cfg).Error; createErr != nil {
			return cfg, createErr
		}
		err = nil
	}
	original := cfg
	NormalizeAIConfig(&cfg)
	if err == nil && (cfg.Provider != original.Provider || cfg.BaseURL != original.BaseURL || cfg.ModelName != original.ModelName) {
		if updateErr := database.Model(&cfg).Updates(map[string]interface{}{"provider": cfg.Provider, "base_url": cfg.BaseURL, "model_name": cfg.ModelName}).Error; updateErr != nil {
			return cfg, updateErr
		}
	}
	return cfg, err
}
