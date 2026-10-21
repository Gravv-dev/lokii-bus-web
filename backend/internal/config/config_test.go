package config

import "testing"

func setBaseEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://app:secret@localhost:5432/cookies?sslmode=disable")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_DOMAIN", "")
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "")
	t.Setenv("HTTP_ADDRESS", "")
}

func TestLoadRequiresExplicitValidCORSOrigin(t *testing.T) {
	setBaseEnvironment(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted a wildcard CORS origin")
	}
}

func TestLoadRejectsPartialBootstrapCredentials(t *testing.T) {
	setBaseEnvironment(t)
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "admin@example.com")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted an email without a bootstrap password")
	}
}

func TestLoadAcceptsLocalConfiguration(t *testing.T) {
	setBaseEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddress != ":8080" || cfg.CookieSecure || len(cfg.AllowedOrigins) != 1 {
		t.Fatalf("unexpected loaded config: %+v", cfg)
	}
}
