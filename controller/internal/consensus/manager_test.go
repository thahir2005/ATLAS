package consensus

import (
	"errors"
	"math"
	"testing"
)

func TestFormatToken(t *testing.T) {
	tests := []struct {
		name  string
		token int64
		want  string
	}{
		{"one digit", 1, "00000000000000000001"},
		{"two digits", 99, "00000000000000000099"},
		{"three digits", 100, "00000000000000000100"},
		{"maximum int64", math.MaxInt64, "09223372036854775807"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatToken(tt.token); got != tt.want {
				t.Fatalf("formatToken(%d) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestFencingTokensPreserveNumericOrder(t *testing.T) {
	tokens := []int64{1, 9, 10, 99, 100, 1000, math.MaxInt64}

	for i := 1; i < len(tokens); i++ {
		previous := formatToken(tokens[i-1])
		current := formatToken(tokens[i])

		if previous >= current {
			t.Fatalf("token order incorrect: %q should be less than %q",
				previous, current)
		}
	}
}

func TestNormalizeConfigDefaults(t *testing.T) {
	cfg, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.TTL != DefaultTTL {
		t.Errorf("TTL = %d, want %d", cfg.TTL, DefaultTTL)
	}
	if cfg.LockKey != DefaultLockKey {
		t.Errorf("LockKey = %q, want %q", cfg.LockKey, DefaultLockKey)
	}
	if cfg.FencingKey != DefaultFenceKey {
		t.Errorf("FencingKey = %q, want %q", cfg.FencingKey, DefaultFenceKey)
	}
}

func TestNormalizeConfigRejectsIdenticalKeys(t *testing.T) {
	_, err := normalizeConfig(Config{
		LockKey:    "/atlas/test",
		FencingKey: "/atlas/test",
	})

	if err == nil {
		t.Fatal("expected error for identical keys")
	}
}

func TestNormalizeConfigPreservesCustomValues(t *testing.T) {
	cfg, err := normalizeConfig(Config{
		TTL:        15,
		LockKey:    "/custom/lock",
		FencingKey: "/custom/fence",
	})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.TTL != 15 || cfg.LockKey != "/custom/lock" ||
		cfg.FencingKey != "/custom/fence" {
		t.Fatalf("custom configuration was not preserved: %+v", cfg)
	}
}

func TestValidateProtectedKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want error
	}{
		{"valid resource", "/atlas/resources/database", nil},
		{"empty key", "", ErrInvalidKey},
		{"lock key", DefaultLockKey, ErrInvalidKey},
		{"lock child", DefaultLockKey + "/owner", ErrInvalidKey},
		{"fencing key", DefaultFenceKey, ErrInvalidKey},
		{"fencing child", DefaultFenceKey + "/child", ErrInvalidKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProtectedKey(tt.key, DefaultLockKey, DefaultFenceKey)

			if !errors.Is(err, tt.want) {
				t.Fatalf("got error %v, want %v", err, tt.want)
			}
		})
	}
}
