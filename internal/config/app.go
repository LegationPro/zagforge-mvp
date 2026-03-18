package config

import (
	"os"
	"strconv"
)

type AppConfig struct {
	GithubAppID            int64
	GithubAppPrivateKey    string
	GithubAppWebhookSecret string
}

func LoadAppConfig() (*AppConfig, error) {
	appIDStr := os.Getenv("GITHUB_APP_ID")
	if appIDStr == "" {
		return nil, notSetErr("GITHUB_APP_ID")
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, notSetErr("GITHUB_APP_ID")
	}

	webhookSecret := os.Getenv("GITHUB_APP_WEBHOOK_SECRET")
	if webhookSecret == "" {
		return nil, notSetErr("GITHUB_APP_WEBHOOK_SECRET")
	}

	privateKeyStr := os.Getenv("GITHUB_APP_PRIVATE_KEY")
	if privateKeyStr == "" {
		return nil, notSetErr("GITHUB_APP_PRIVATE_KEY")
	}

	return &AppConfig{
		GithubAppID:            appID,
		GithubAppPrivateKey:    privateKeyStr,
		GithubAppWebhookSecret: webhookSecret,
	}, nil
}
