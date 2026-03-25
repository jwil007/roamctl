# roamctl
roamctl is a Linux utility that provides a fully configurable Wi-Fi roaming algorithm. It is written in Go, and exclusively utilizes the wpa_supplicant control interface for Wi-Fi operations. For more info on the wpa_supplicant control interface, read the official docs https://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html.

While running, roamctl disables wpa_supplicant's autonomous roaming and instead uses a configurable roaming algorithm. The algorithm is score based, using a method to score each BSSID in the scan data to make a decision whether or not to roam (reassociate). When the program exits, the original wpa_supplicant configuration and roaming behavior is restored.

The configurable roaming algorithm allows full control of the roaming behavior, and can be used to test unique scenarios.  For example, if you set phy_scores to `80211ac = 100` and `80211ax = 20`, the device will strongly prefer Wi-Fi 5 APs over Wi-Fi 6.  See [Configuration](#Configuration) for full parameter reference.

There are also templates designed to map to real world devices. Currently, templates are available to simulate Apple MacOS and iOS roaming, based on their documented algorithm.

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
The roaming algorithm consists continuous signal polling, concurrent background scanning, and four different tiers based on signal metrics which determine roaming behavior.

### Roaming Tiers
There are four tiers which set roaming behavior, such as aggresiveness, to allow for perfomant roaming in a wide variety of environments. The tiers are based on RSSI breakpoints, which are user configurable.

#### Excellent - No roaming or scanning
In the Excellent tier, background scanning is paused and no roaming will occur.

#### Fair - Passive roaming / background scan
In the Fair tier, backgound scanning begins using a smart scan to only scan relevant channels. Background scan results are evaluated, and if an AP is significantly better than the current AP, a roam will occur.

#### Degraded - Active roaming / full scan if needed
In the Degraded tier, active roaming begins. When this tier is entered a scan on targeted channels is performed to see if a better AP is available. If a better AP is not found, a full scan is requested for the next background scan interval.

#### Critical - Aggressive roaming
When the critical tier is entered, another immediate fast scan is triggered. If no better APs are found, a full scan is run in line, rather than waiting for the next background scan interval. The score delta required to roam is lower in this tier by default.
### Visual Diagram
A flow chart diagram is avaialble for a simplified view at the roaming algorithm.
- [Algorithm flowchart](/docs/algorithm-chart.md)

### Scoring
BSSIDs in the scan results are scored using a weighted combination of RSSI, SNR, Band, channel utilization, PHY type, etc. The scoring parameters and weights are user-configurable, See [Configuration](#Configuration).

### Scanning and Stability 
A number of stability guards are in place to prevent excessive roaming, scanning or ping-ponging. These guards include:
- A score_delta parameter to ensure that the candidate AP is materially better than the current AP
- A smart scanning algorithm is used to minimze time spent scanning, by scanning only channels known to have candidate APs. Full sweeps are only done when needed.
- [Hysteresis methods](https://en.wikipedia.org/wiki/Hysteresis#Control_systems) to prevent freqently entering the roam cycle when at a borderline signal strength

> [!IMPORTANT]
> While effort has been made to ensure stability in various edge cases, this is not a battle-tested roaming algorithm. It is meant primarily to be a tool for Wi-Fi engineers, allowing easy access to test and simulate client behavior.

## Usage
Connect to an SSID. Run with `roamctl`. Exit with `ctrl+c`.

Some Linux distros require elevated permissions.

If you get a permission error, run with `sudo`. 

### Flags:

`-edit` : Edit config file. Checks for $EDITOR env variable, otherwise tries nano, then vi.

`-level` : Set log level. Options are `info` or `debug`. Default is `info`

`-template`: Select config template. Options are `base`, `macos`, and `ios`.

## A note on 6GHz
The version of wpa_supplicant that ships with most Debian based distros (v2.10) may not reliably find 6GHz APs in the scan results. If you're testing 6GHz roaming, check your version with `wpa_supplicant -v`. If you have v2.11 or newer, 6GHz should be reliable.

Upgrading from v2.10 to v2.11 on Debian based distros may require building from source. This GitHub repo contains a shell script to handle the build process for you: https://github.com/jwil007/wpas-v2.11-install-script/tree/main.

>[!WARNING]
> The script replaces the distro-packaged wpa_supplicant binary in-place without going through the package manager. Future apt upgrade runs may overwrite it with the older distro version.
>
> The script will restart wpa_supplicant when it is completed. This briefly disconnects Wi-Fi, which means that you may lose connection to the machine if accessing it remotely and it is only Wi-Fi connected.


## Configuration

All config parameters, including interface specification and scoring weights for the roaming algorithm, are set through the toml file at `~/.config/roamctl/config.toml`

For convenience, running with the `-edit` flag opens a text editor to edit the file directly.

The `-template` flag allows selection of pre-defined templates. The current options are `base` (default), `macos`, and `ios`.

The MacOS and iOS templates are meant to simulate Apple roaming behavior as documented in: https://support.apple.com/guide/deployment/wi-fi-roaming-support-dep98f116c0f/web

>[!NOTE]
>roamctl initializes with the params shown in the default config below.

## Default Config
```toml
[preferences]
interface = "wlan0"

[roaming_tiers]
# These values set the floor of each RSSI tier, which dictate roaming logic
# They are the floor for each tier. 
# i.e. if fair_rssi is -67, -68 is in the "degraded" tier.
excellent_rssi = -58
fair_rssi = -70
degraded_rssi = -76
# values lower than degraded are considered critical

# Set score deltas required to roam per tier
# Higher numbers mean candidate AP must be significantly better
fair_score_delta = 8
degraded_score_delta = 5
critical_score_delta = 3

[stability]
# These values define upper and lower bounds RSSI must cross before re-roaming
# This is meant to prevent ping-ponging when at RSSI boundry
rssi_hysteresis_up = 5
rssi_hysteresis_down = 5

[timing]
# Timings for signal polling (e.g. hosts current RSSI, SNR) and bg scan interval
sig_poll_interval = "250ms"
base_scan_interval = "15s"

# Defines number of bssids used to build fast-scan channel list
# In very dense environments, this number can be tuned to optimize channel scanning
max_bss_ct = 15

[score_weights]
# These tune the scoring algorithm, which ranks candidate APs
# Set to 100 for maxmium weight, 0 to ignore category
rssi = 100
snr = 0
qbss_util = 30
band = 50
channel_width = 10
phy_type = 15

[score_clamps]
# Used for RSSI and SNR scoring.
# Values below min are scored 0, values above max are score 100
# Values between clamps are scored linearly
min_rssi = -85
max_rssi = -25
min_snr = 10
max_snr = 50

# Use the following to adjust various scoring. This is where you can 
# tweak band pref, cw pref, etc.
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
```
