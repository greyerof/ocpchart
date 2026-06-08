package config

import (
	"testing"
	"time"
)

func TestValidateAuthFlags_KubeconfigOnly(t *testing.T) {
	if err := ValidateAuthFlags("/path/to/kubeconfig", "", ""); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateAuthFlags_ManualTokenAndURL(t *testing.T) {
	if err := ValidateAuthFlags("", "https://thanos.example.com", "mytoken"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateAuthFlags_KubeconfigWithManual(t *testing.T) {
	err := ValidateAuthFlags("/path", "https://thanos", "tok")
	if err == nil {
		t.Fatal("expected error when combining kubeconfig with manual flags")
	}
}

func TestValidateAuthFlags_TokenWithoutURL(t *testing.T) {
	err := ValidateAuthFlags("", "", "mytoken")
	if err == nil {
		t.Fatal("expected error when token is provided without URL")
	}
}

func TestValidateAuthFlags_URLWithoutToken(t *testing.T) {
	err := ValidateAuthFlags("", "https://thanos.example.com", "")
	if err == nil {
		t.Fatal("expected error when URL is provided without token")
	}
}

func TestValidateAuthFlags_NothingProvided(t *testing.T) {
	t.Setenv("KUBECONFIG", "")
	err := ValidateAuthFlags("", "", "")
	if err == nil {
		t.Fatal("expected error when nothing is provided")
	}
}

func TestValidateAuthFlags_FallbackToEnvKubeconfig(t *testing.T) {
	t.Setenv("KUBECONFIG", "/some/path")
	if err := ValidateAuthFlags("", "", ""); err != nil {
		t.Fatalf("expected no error with KUBECONFIG env, got: %v", err)
	}
}

func TestAutoStep_WithExplicitWidth(t *testing.T) {
	step := AutoStep(time.Hour, 100)
	// 100 - 10 = 90 points -> 3600s / 90 = 40s
	if step != 40*time.Second {
		t.Fatalf("expected 40s step, got %s", step)
	}
}

func TestAutoStep_ClampsToMinimum(t *testing.T) {
	// Very short duration should clamp to MinStep
	step := AutoStep(30*time.Second, 100)
	if step != MinStep {
		t.Fatalf("expected MinStep (%s), got %s", MinStep, step)
	}
}

func TestAutoStep_SmallWidth(t *testing.T) {
	// Width 15 -> points = max(20, 15-10) = 20
	step := AutoStep(time.Hour, 15)
	expected := time.Hour / 20
	if step != expected {
		t.Fatalf("expected %s, got %s", expected, step)
	}
}

func TestAutoStep_ZeroWidth_UsesTerminal(t *testing.T) {
	// With width 0, it falls back to TerminalWidth() which in CI returns 80
	step := AutoStep(time.Hour, 0)
	if step < MinStep {
		t.Fatalf("step should be >= MinStep, got %s", step)
	}

	if step > time.Hour {
		t.Fatalf("step should be <= since, got %s", step)
	}
}
