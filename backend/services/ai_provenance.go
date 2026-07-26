package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"study_plan_backend/config"
)

type PlanProvenanceClaims struct {
	UserID             uint     `json:"user_id"`
	Source             string   `json:"source"`
	PreviewID          string   `json:"preview_id,omitempty"`
	PreviewVersion     int      `json:"preview_version,omitempty"`
	ContextFingerprint string   `json:"context_fingerprint,omitempty"`
	ImmutableHash      string   `json:"immutable_hash"`
	TaskIdentities     []string `json:"task_identities"`
	Type               string   `json:"type"`
	jwt.RegisteredClaims
}

func HashPlanPreview(preview PlanPreview) string {
	payload, _ := json.Marshal(preview)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func SignPlanProvenance(userID uint, source string, preview PlanPreview) (string, error) {
	return signPlanProvenance(userID, source, "", 0, "", time.Now().Add(30*time.Minute), preview)
}

func SignPlanVersionProvenance(userID uint, source, previewID string, version int, contextFingerprint string, expiresAt time.Time, preview PlanPreview) (string, error) {
	if previewID == "" || version < 1 || contextFingerprint == "" {
		return "", errors.New("preview version metadata is required")
	}
	return signPlanProvenance(userID, source, previewID, version, contextFingerprint, expiresAt, preview)
}

func signPlanProvenance(userID uint, source, previewID string, version int, contextFingerprint string, expiresAt time.Time, preview PlanPreview) (string, error) {
	if source != "local" && source != "local_enriched" && source != "ai_decomposed" {
		return "", errors.New("invalid generation source")
	}
	now := time.Now()
	identities := make([]string, len(preview.Tasks))
	for index, task := range preview.Tasks {
		if task.Identity == "" {
			return "", errors.New("preview task identity is required")
		}
		identities[index] = task.Identity
	}
	claims := PlanProvenanceClaims{UserID: userID, Source: source, PreviewID: previewID, PreviewVersion: version, ContextFingerprint: contextFingerprint, ImmutableHash: HashPlanPreviewImmutable(preview), TaskIdentities: identities, Type: "plan_preview", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiresAt), IssuedAt: jwt.NewNumericDate(now), Issuer: "study_plan_preview"}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.App.JWTSecret))
}

func ParsePlanProvenance(token string, userID uint) (*PlanProvenanceClaims, error) {
	claims := &PlanProvenanceClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(config.App.JWTSecret), nil
	})
	if err != nil || !parsed.Valid || claims.Type != "plan_preview" || claims.Issuer != "study_plan_preview" || claims.UserID != userID || claims.ImmutableHash == "" || len(claims.TaskIdentities) == 0 || (claims.Source != "local" && claims.Source != "local_enriched" && claims.Source != "ai_decomposed") {
		return nil, errors.New("invalid or expired preview provenance")
	}
	return claims, nil
}

func HashPlanPreviewImmutable(preview PlanPreview) string {
	payload, _ := json.Marshal(struct {
		Title     string `json:"title"`
		Summary   string `json:"summary"`
		Rationale string `json:"rationale"`
		TaskCount int    `json:"task_count"`
	}{preview.Title, preview.Summary, preview.Rationale, len(preview.Tasks)})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func ValidateCommittedPlanProvenance(preview PlanPreview, claims *PlanProvenanceClaims) error {
	if HashPlanPreviewImmutable(preview) != claims.ImmutableHash || len(preview.Tasks) != len(claims.TaskIdentities) {
		return errors.New("preview provenance does not match generated candidate")
	}
	for index, task := range preview.Tasks {
		if task.Identity != claims.TaskIdentities[index] {
			return errors.New("preview task identity or order changed")
		}
	}
	return nil
}
