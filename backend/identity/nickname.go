package identity

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var contactPattern = regexp.MustCompile(`(?i)(1[3-9][0-9]{9}|[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}|(?:微信|wx|qq)[:： ]?[a-z0-9_-]{5,})`)

func Normalize(value string) string {
	return strings.ToLower(norm.NFKC.String(strings.TrimSpace(value)))
}

func NewInviteTargetID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func Validate(value string) (string, string, error) {
	display := norm.NFKC.String(strings.TrimSpace(value))
	key := Normalize(display)
	count := utf8.RuneCountInString(display)
	if count < 2 || count > 20 {
		return "", "", errors.New("nickname must contain 2-20 characters")
	}
	lettersOrNumbers := false
	for _, r := range display {
		if unicode.IsControl(r) {
			return "", "", errors.New("nickname cannot contain control characters")
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			lettersOrNumbers = true
		}
	}
	if !lettersOrNumbers {
		return "", "", errors.New("nickname must contain a letter or number")
	}
	reserved := []string{"admin", "administrator", "管理员", "系统", "官方", "客服"}
	for _, word := range reserved {
		if strings.Contains(key, word) {
			return "", "", errors.New("nickname contains a reserved term")
		}
	}
	if contactPattern.MatchString(display) {
		return "", "", errors.New("nickname cannot contain contact information")
	}
	return display, key, nil
}
