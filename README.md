# Frigate-Notify

<div align="center">

[![Static Badge](https://img.shields.io/badge/Documentation-blue)](https://frigate-notify.0x2142.com) &nbsp;&nbsp; [![GitHub Repo stars](https://img.shields.io/github/stars/0x2142/frigate-notify)]() &nbsp;&nbsp; [![GitHub release (with filter)](https://img.shields.io/github/v/release/0x2142/frigate-notify)](https://github.com/0x2142/frigate-notify/releases) &nbsp;&nbsp; [![Static Badge](https://img.shields.io/badge/Docker-latest-blue)](https://github.com/0x2142/frigate-notify/pkgs/container/frigate-notify) &nbsp;&nbsp; [![BuyMeACoffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-blue?style=flat&logo=buy-me-a-coffee&logoColor=white)](https://www.buymeacoffee.com/0x2142)

</div>

## About

Frigate-Notify is a simple app designed to send notifications from [Frigate](https://github.com/blakeblackshear/frigate) NVR to your favorite platforms. Intended to be used with standalone Frigate installations - Home Assistant not required, MQTT is optional but recommended.

## Fork: Event Notification Cooldowns

This repository is a source-based fork of [0x2142/frigate-notify](https://github.com/0x2142/frigate-notify). The original project, licensing, and documentation remain credited to its maintainer. This fork adds the scope-aware cooldown feature proposed in [issue #264](https://github.com/0x2142/frigate-notify/issues/264#issuecomment-4000094887).

For **Frigate 0.13.2**, use `app.mode: events` (there are no Frigate Reviews in that release). Set one or more cooldowns in `config.yml`:

```yaml
app:
  mode: events

alerts:
  notify_limits:
    # global: 15s
    camera:
      cam_door: 1m
    label:
      car: 5m
    zone:
      front_door: 2m
```

The first matching event sends an alert and reserves every applicable scope. During a cooldown, further alerts matching **any** active rule are suppressed and do not extend the timers. A label cooldown applies across **all cameras**; zones are matched by their exact name. Values use Go durations, such as `30s`, `5m`, or `1h`. All rules are **disabled by default**. Settings are documented in [docs/config/file.md](docs/config/file.md#notification-cooldowns-fork-extension).

**Caveats:** A cooldown can hide a later important event. Suppressed events are not queued or summarized. State resets upon service restart. Manual test alerts bypass cooldowns. Docker images from upstream do not contain this fork's changes; build this repository's Dockerfile instead.

## Features

### Event Polling

- MQTT
- Direct via Frigate API

### Notification Methods

- Discord
- Gotify
- Matrix
- Mattermost
- Ntfy
- Pushover
- Signal
- SMTP
- Telegram
- Webhook
- And many others via the [Apprise API](https://github.com/caronc/apprise-api)

### Other

- Aliveness monitor via HTTP GET (for use with tools like [HealthChecks](https://github.com/healthchecks/healthchecks) or [Uptime Kuma](https://github.com/louislam/uptime-kuma))

## Installation

See [Install / Setup](https://frigate-notify.0x2142.com/latest/install/).

## Configuration

An example config file has been provided (`example-config.yml`). Please copy this & edit to fit your needs.

The sample config contains inline descriptions for each field. For additional details on the available config options & the defaults, see the [Config](https://frigate-notify.0x2142.com/latest/config/) docs.

## Screenshots

For example screenshots of app notifications, see [here](https://github.com/0x2142/frigate-notify/tree/main/screenshots)

## Changelog

Additional details on each release are in the [Changelog](https://frigate-notify.0x2142.com/latest/changelog/)

## Support

<a href="https://www.buymeacoffee.com/0x2142" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-blue.png" alt="Buy Me A Coffee" height="41" width="174"></a>
