# roamctl
roamctl is a Linux utility that provides a fully configurable Wi-Fi roaming algorithm. It is written in Go, and exclusively utilizes the wpa_supplicant control interface for Wi-Fi operations. For more info on the wpa_supplicant control interface, read the official docs https://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html.

While running, roamctl disables wpa_supplicant's autonomous roaming and instead uses a configurable roaming algorithm. The algorithm is score based, using a method to score each BSSID in the scan data to make a decision whether or not to roam (reassociate). When the program exits, the original wpa_supplicant configuration and roaming behavior is restored.

The configurable roaming algorithm allows simulation of various client behavior. For example, if you adjust the `band_scores` params to `2point4ghz = 100`, `5ghz = 25`, and `6ghz = 15`, the device will strongly prefer 2.4GHz over 5GHz and 6GHz, and slightly prefer 5GHz over 6GHz. There are over 30 parameters that can be set. See [Configuration](#Configuration) for full parameter reference.

Output is shown in the terminal using structured logging with timestamps and log levels.

## Quick Start

### One-line install
Automatically downloads and installs the Linux binary for AMD64, ARM64, or ARM32 devices. 
```
curl -fsSL https://raw.githubusercontent.com/jwil007/roamctl/master/install.sh | bash
```
### Build from source
<details>
  <summary>Click to expand</summary>
  Make sure you have installed Go for Linux: https://go.dev/doc/install.

Builds and installs to `$GOPATH/bin`. Most likely this is `~/go/bin`. 

```
go install github.com/jwil007/roamctl/cmd/roamctl@latest
```
 
If running `roamctl` after the Go install returns "command not found", make sure the go/bin directory is in your path.

The command below will add the path config for your default shell. After running the command, restart your shell session and you should be able to run `roamctl`.
```
 echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.$(basename $SHELL)rc
 ```

</details>

## Algorithm details
The roaming algorithm consists of an outer loop for polling and an inner tree for roaming decisions.

The outer loop handles polling to watch client metrics, such as RSSI, noise, and data rate. it also runs a background scan on a configurable time interval.

The inner tree is reached when a threshold, such as RSSI, is below a set value. At that point the inner tree evaluates scan results against the current connected AP and decides whether or not to roam.

### Visual diagrams
Flow chart diagrams are avaialble for both the outer loop algorithm, and the roam decision tree.
- [Outer Loop Algorithm](/docs/algorithm-chart.md)
- [Roam Decision Tree](/docs/roam-decision-chart.md)

### Scoring and Stability
BSSIDs in the scan results are scored using a weighted combination of RSSI, SNR, Band, channel utilization, PHY type, etc. The scoring parameters and weights are user-configurable, See [Configuration](#Configuration).

A number of stability guards are in place to prevent excessive roaming, scanning or ping-ponging. These guards include:
- A minimum score_delta parameter, make sure the candidate AP is materially better than the current AP.
- Backoff timers after the roam cycle
- Fallback to passive roaming if no acceptable candidate APs seen after multiple attempts
- [Hysteresis methods](https://en.wikipedia.org/wiki/Hysteresis#Control_systems) to prevent freqently entering the roam cycle when at a borderline signal strength

> [!IMPORTANT]
> While effort has been made to ensure stability in various edge cases, this is not a battle-tested roaming algorithm. It is meant primarily to be a tool for Wi-Fi engineers, allowing easy access to test and simulate client behavior.

## Usage
Connect to an SSID. Run with `roamctl`. Exit with `ctrl+c`.

Some Linux distros require elevated permissions.

If you get a permission error, run with `sudo`. 

#### Flags:

`-edit` : Edit config file. Checks for $EDITOR env variable, otherwise tries nano, then vi.

`-reset` : Reset config file to defaault template.

`-level` : Set log level. Options are `info` or `debug`. Default is `info`

## Configuration

All config parameters, including interface specification and scoring weights for the roaming algorithm, are set through the toml file at `~/.config/roamctl/config.toml`

For convenience, running with the `-edit` flag opens a text editor to edit the file directly.

The `-reset` flag overwrites the config file with the default template.

>[!NOTE]
>roamctl initializes with "sensible default" parameters. The Default Config shown below.

## Default Config
```toml
[preferences]
# Set Wi-Fi interface name
  interface = "wlan0"

[thresholds]
# Thresholds from signal polling which define when to enter
# the roaming decision loop.
# For example, if the rssi threshold is -67, the device will enter
# the roam decision loop when RSSI is -68 or lower.
  rssi = -67 # dBm, allowed range -128 to 0

# Set upper and lower bounds for the RSSI hysteresis band in dBm
# Hysteresis is activated after a roam attempt with no candidates
# RSSI must leave the hysteresis band before the roam loop is re-entered
# Must be integer in range 0 to 15
  rssi_hysteresis_up = 3
  rssi_hysteresis_down = 5

# Set data rate to 0 to ignore, otherwise set value as Mbps.
# Roam decision loop entered when polled data rate < threshold
  data_rate = 0 # Mbps

# score_delta score difference required to roam to a new AP.
# Lower values: more roaming, less stable
# Must be integer in range 0 to 100
  score_delta = 10

# max_no_candidate_attempts defines the max number of consecutive
# roam attempts where no candidate is found before falling back 
# to bgscan monitoring.
# Must be integer in range 0 to 20
  max_no_candidate_attempts = 3

[score_weights]
# Score weights are a multipier on each scoring category
# A value of 0 means a category is ignored from the scoring algorithm.
# 100 is the max value.
  rssi = 100
  snr = 50

# qbss utilzation parsed from beacon frames. Akin to channel utilzation
  qbss_util = 25

# Weights for band (2.4/5/6GHz), chan width, and PHY type (wifi version)
# Scores for each value within these categories are defined below
  band = 60
  channel_width = 0
  phy_type = 25

[score_clamps]
# Min and max RSSI used to clamp scoring algorithm.
# Values below min are scored 0, values above max are scored 100.
  min_rssi = -85
  max_rssi = -30

# Min and max SNR used to clamp scoring algorithm.
# Values below min are scored 0, values above max are scored 100
  min_snr = 10
  max_snr = 50

[band_scores]
# All scores below must be integers 0 to 100
  2point4ghz = 0
  5ghz = 80
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

[timing]
# Times must use the format ms for millisecond, s for second, m for minute
# Amount of time to wait before re-enterting roam loop depending on outcome
  success_backoff_time = "2s"
  failure_backoff_time = "2s"
  no_candidates_backoff_time = "3s"

# Defines how often signal metrics for roaming threshold are checked.
  sig_poll_interval = "250ms"

# When not in roam decision loop, define how frequently wifi scan is done
  bg_scan_interval = "30s"

# A candidate AP in the scan data must be "newer" than the max_scan_age
# to be considered.
  max_scan_age = "10s"
```
