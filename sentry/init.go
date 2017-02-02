package sentry

import (
	log "github.com/Sirupsen/logrus"
	"github.com/evalphobia/logrus_sentry"
	"github.com/getsentry/raven-go"
)

func Init(config Config) error {
	if config.DSN == "" {
		log.Info("Sentry DSN is not configured - Sentry will be disabled")
		return nil
	}

	client, err := raven.New(config.DSN)
	if err != nil {
		return err
	}
	client.SetEnvironment(config.Env)
	client.SetRelease(config.Release)

	levels, err := parseLogLevels(config)
	if err != nil {
		return err
	}

	sentryHook, err := logrus_sentry.NewWithClientSentryHook(client, levels)
	if err != nil {
		return err
	}
	log.AddHook(sentryHook)
	log.Infof("Enabled Sentry alerting for following logging levels: %v", levels)

	return nil
}

func parseLogLevels(config Config) ([]log.Level, error) {
	boundLevel, err := log.ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}

	var levels []log.Level

	for _, level := range log.AllLevels {
		levels = append(levels, level)
		if level == boundLevel {
			break
		}
	}

	return levels, nil
}
