# roamctl
This is a Linux utility with the goal of providing a fully configurable Wi-Fi roaming algorithm. It is written in Go, and exclusively utilizes the wpa_supplicant control interface (https://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html) for all Wi-Fi operations.

The program works by first disabling wpa_supplicant's autonomous roaming, and then utilizing a configurable algorithm, which primarily uses a per-BSSID scoring method to make roaming decisions. When the program exists, the devices original wpa_supplicant configuration is restored.

Useful primitives in the ctrl interface such as `ROAM`, `SCAN`, `SIGNAL_POLL`, and `SCAN_RESULTS` make this type of program possible.

Output is logged to terminal with microsecond timestamp precision and color coding for important events.

> [!IMPORTANT]
> This is not a battle tested roaming algorithm. It is meant primarily to be a testing tool for Wi-Fi tinkerers.

## Quick Start
TODO

## Usage
Connect to an SSID. Run with `./roamctl.` Exit with `ctrl+c`.

#### Flags:

`-edit` : Edit config file. Checks for $EDITOR env variable, otherwise tries nano, then vi.

`-reset` : Reset config file to original 

## Configuration
All config parameters, including interface specification and scoring weights for the roaming algoithm, are set through the config.toml file at `/home/USER/.config/roamctl/config.toml`

For convenience, running with the `-edit` flag opens a text editor to edit the file directly. The `-reset` flag overwrites the config file with the default template.

## Default Config
```toml
[preferences]
# Set Wi-Fi interface name
interface = "wlan0"

[thresholds]
# Thesholds from signal polling which define when to enter
# the roaming decision loop.
# For example, if the rssi threshold is -67, the device will enter
# the roam decision loop when RSSI is -68 or lower.
rssi = -67 # dBm, allowed range -128 to 0
data_rate = 54 # mbps

# score_delta score difference required to roam to a new AP.
# Lower values: more roaming, less stable
# Must be integer in range 0 to 100
score_delta = 7

[score_weights]
# Score weights are a multipier on each scoring category
# A value of 0 means a category is ignored from the scoring algorithm.
# 100 is the max value.
rssi = 100

# Min and max RSSI used to clamp scoring algorithm.
# Values below min are scored 0, values above max are scored 100.
min_rssi = -80
max_rssi = -40

snr = 50
# Min and max SNR used to clamp scoring algorithm.
# Values below min are scored 0, values above max are scored 100
min_snr = 10
max_snr = 50

# qbss utilzation parsed from beacon frames. Akin to channel utilzation
qbss_util = 25

# Weights for band (2.4/5/6GHz), chan width, and PHY type (wifi version)
# Scores for each value within these categories are defined below
band = 50
channel_width = 10
phy_type = 25

[timing]
# Times must use the format ms for millisecond, s for second, m for minute
# Amount of time to wait before re-enterting roam loop depending on outcome
success_backoff_time = "5s"
failure_backoff_time = "2s"
no_candidates_backoff_time = "7s"

# Defines how often signal metrics for roaming threshold are checked.
sig_poll_interval = "300ms"

# When not in roam decision loop, define how frequently wifi scan is done
bg_scan_interval = "30s"

# A candidate AP in the scan data must be "newer" than the max_scan_age
# to be considered.
max_scan_age = "10s"

# All scores below must be integers 0 to 100
[band_scores]
2point4ghz = 0
5ghz = 85
6ghz = 100

[chan_width_scores]
20mhz = 30
40mhz = 60
80mhz = 80
160mhz = 90
320mhz = 100

[phy_scores]
legacy = 0 # Wi-Fi 3 or older
80211n = 20 # Wi-Fi 4
80211ac = 50 # Wi-Fi 5
80211ax = 80 # Wi-Fi 6
80211be = 100 # Wi-Fi 7
```
