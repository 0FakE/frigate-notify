package notifier

import (
	"github.com/0x2142/frigate-notify/config"
	"github.com/0x2142/frigate-notify/models"
)

// hasEligibleAlertProvider ensures that provider-filtered alerts never
// reserve cooldown slots. This check happens after snapshot retrieval.
func hasEligibleAlertProvider(events []models.Event) bool {
	for id, profile := range config.ConfigData.Alerts.AppriseAPI {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "apprise_api", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Discord {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "discord", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Gotify {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "gotify", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Matrix {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "matrix", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Mattermost {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "mattermost", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Ntfy {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "ntfy", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Pushover {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "pushover", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Signal {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "signal", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.SMTP {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "smtp", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Telegram {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "telegram", index: id}) {
			return true
		}
	}
	for id, profile := range config.ConfigData.Alerts.Webhook {
		if profile.Enabled && checkAlertFilters(events, profile.Filters, notifMeta{name: "webhook", index: id}) {
			return true
		}
	}
	return false
}
