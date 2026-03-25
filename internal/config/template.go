package config

const defaultConfigTemplate = `
[preferences]
interface = "wlan0"

[roaming_tiers]
excellent_rssi = -50
opportunistic_rssi = -65
opportunistic_score_delta = 10
active_rssi = -71
active_score_delta = 7
critical_rssi = -76
critical_score_delta = 3

[stability]
rssi_hysteresis_up = 5
rssi_hysteresis_down = 5

[timing]
sig_poll_interval = "100ms"
base_scan_interval = "15s"
max_bss_ct = 15

[score_weights]
rssi = 100
snr = 0
qbss_util = 30
band = 50
channel_width = 10
phy_type = 15

[score_clamps]
min_rssi = -85
max_rssi = -25
min_snr = 10
max_snr = 50

[band_scores]
2point4ghz = 0
5ghz = 75
6ghz = 100

[chan_width_scores]
20mhz = 0
40mhz = 25
80mhz = 75
160mhz = 90
320mhz = 100

[phy_scores]
legacy = 0
80211n = 20
80211ac = 50
80211ax = 80
80211be = 100
`

const macOSTemplate = `
# This template aims to simulate documented Apple macOS roaming behavior:
# https://support.apple.com/guide/deployment/wi-fi-roaming-support-dep98f116c0f/web

[preferences]
  interface = "wlan0"

[roaming_tiers]
  excellent_rssi = -60
  opportunistic_rssi = -67
  opportunistic_score_delta = 15
  active_rssi = -75
  active_score_delta = 12
  critical_rssi = -82
  critical_score_delta = 6

[stability]
  rssi_hysteresis_up = 8
  rssi_hysteresis_down = 5

[timing]
  sig_poll_interval = "250ms"
  base_scan_interval = "30s"
  max_bss_ct = 15

[score_weights]
  rssi = 100
  snr = 25
  qbss_util = 40
  band = 50
  channel_width = 30
  phy_type = 30

[score_clamps]
  min_rssi = -85
  max_rssi = -30
  min_snr = 10
  max_snr = 50

[band_scores]
  2point4ghz = 10
  5ghz = 80
  6ghz = 100

[chan_width_scores]
  20mhz = 20
  40mhz = 50
  80mhz = 75
  160mhz = 95
  320mhz = 100

[phy_scores]
  legacy = 0
  80211n = 15
  80211ac = 50
  80211ax = 80
  80211be = 100
`

const iOSTemplate = `
# This template aims to simulate documented Apple iOS/iPadOS roaming behavior:
# https://support.apple.com/guide/deployment/wi-fi-roaming-support-dep98f116c0f/web

[preferences]
  interface = "wlan0"

[roaming_tiers]
  excellent_rssi = -60
  opportunistic_rssi = -63
  opportunistic_score_delta = 12
  active_rssi = -70
  active_score_delta = 8
  critical_rssi = -78
  critical_score_delta = 5

[stability]
  rssi_hysteresis_up = 6
  rssi_hysteresis_down = 4

[timing]
  sig_poll_interval = "250ms"
  base_scan_interval = "20s"
  base_scan_interval = "20s"
  max_bss_ct = 15

[score_weights]
  rssi = 100
  snr = 25
  qbss_util = 40
  band = 50
  channel_width = 30
  phy_type = 30

[score_clamps]
  min_rssi = -85
  max_rssi = -30
  min_snr = 10
  max_snr = 50

[band_scores]
  2point4ghz = 10
  5ghz = 80
  6ghz = 100

[chan_width_scores]
  20mhz = 20
  40mhz = 50
  80mhz = 75
  160mhz = 95
  320mhz = 100

[phy_scores]
  legacy = 0
  80211n = 15
  80211ac = 50
  80211ax = 80
  80211be = 100
`
