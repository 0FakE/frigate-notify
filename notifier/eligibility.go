package notifier

import (
	"github.com/0x2142/frigate-notify/config"
	"github.com/0x2142/frigate-notify/models"
)

// eligibleAlertProviders ensures that provider-filtered alerts never
// reserve cooldown slots. This check happens after snapshot retrieval.
func eligibleAlertProviders(events []models.Event) map[notifMeta]bool {
	eligible := make(map[notifMeta]bool)
	for id, profile := range config.ConfigData.Alerts.AppriseAPI {
		provider := notifMeta{name: "apprise_api", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Discord {
		provider := notifMeta{name: "discord", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Gotify {
		provider := notifMeta{name: "gotify", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Matrix {
		provider := notifMeta{name: "matrix", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Mattermost {
		provider := notifMeta{name: "mattermost", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Ntfy {
		provider := notifMeta{name: "ntfy", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Pushover {
		provider := notifMeta{name: "pushover", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Signal {
		provider := notifMeta{name: "signal", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.SMTP {
		provider := notifMeta{name: "smtp", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Telegram {
		provider := notifMeta{name: "telegram", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Webhook {
		provider := notifMeta{name: "webhook", index: id}
		if profile.Enabled && checkAlertFilters(events, profile.Filters, provider) {
			eligible[provider] = true
		}
	}
	return eligible
}
