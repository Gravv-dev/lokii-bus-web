package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddress            string
	DatabaseURL            string
	AllowedOrigins         []string
	CookieSecure           bool
	CookieDomain           string
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return Config{}, errors.New("DATABASE_URL must be a valid PostgreSQL URL")
	}

	secure, err := optionalBool("COOKIE_SECURE", true)
	if err != nil {
		return Config{}, err
	}
	adminEmail := strings.TrimSpace(strings.ToLower(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")))
	adminPassword := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if (adminEmail == "") != (adminPassword == "") {
		return Config{}, errors.New("BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD must be set together")
	}
	if adminPassword != "" && (len(adminPassword) < 12 || len(adminPassword) > 72) {
		return Config{}, errors.New("BOOTSTRAP_ADMIN_PASSWORD must be between 12 and 72 bytes")
	}
	if adminEmail != "" && !strings.Contains(adminEmail, "@") {
		return Config{}, errors.New("BOOTSTRAP_ADMIN_EMAIL must be a valid email address")
	}

	origins := make([]string, 0)
	for _, origin := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		parsedOrigin, parseErr := url.Parse(origin)
		if parseErr != nil || parsedOrigin.Host == "" || parsedOrigin.User != nil ||
			parsedOrigin.Path != "" || parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" ||
			(parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https") {
			return Config{}, fmt.Errorf("invalid CORS origin %q", origin)
		}
		origins = append(origins, origin)
	}
	if len(origins) == 0 {
		return Config{}, errors.New("CORS_ALLOWED_ORIGINS must contain at least one explicit origin")
	}

	address := strings.TrimSpace(os.Getenv("HTTP_ADDRESS"))
	if address == "" {
		address = ":8080"
	}
	return Config{
		HTTPAddress:            address,
		DatabaseURL:            databaseURL,
		AllowedOrigins:         origins,
		CookieSecure:           secure,
		CookieDomain:           strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")),
		BootstrapAdminEmail:    adminEmail,
		BootstrapAdminPassword: adminPassword,
	}, nil
}

func optionalBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return parsed, nil
}
