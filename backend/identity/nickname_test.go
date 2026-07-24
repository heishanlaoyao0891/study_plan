package identity

import "testing"

func TestNicknameNormalizationAndValidation(t *testing.T) {
	display, key, err := Validate("  Ａlice  ")
	if err != nil {
		t.Fatal(err)
	}
	if display != "Alice" || key != "alice" {
		t.Fatalf("unexpected normalization: display=%q key=%q", display, key)
	}
	for _, value := range []string{"A", "😀😀", "系统客服", "13800138000"} {
		if _, _, err := Validate(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	first, err := NewInviteTargetID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewInviteTargetID()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 24 || first == second {
		t.Fatalf("invalid opaque ids: %q %q", first, second)
	}
}
