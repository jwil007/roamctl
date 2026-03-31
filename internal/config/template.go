package config

const defaultConfigTemplate = `
[preferences]
interface = "wlan0"

[roaming_tiers]
# These values set the floor of each RSSI tier, which dictate roaming logic
# i.e. if fair_rssi is -67, -68 is in the "degraded" tier.
excellent_rssi = -58
fair_rssi = -67
degraded_rssi = -73
# values lower than degraded are considered critical

# Set score deltas required to roam per tier
# Higher numbers mean candidate AP must be significantly better
fair_score_delta = 9
degraded_score_delta = 8
critical_score_delta = 7

[stability]
# sets cooldown duration after connection changed
# prevents roaming while cooldown is in effect
connection_cooldown = "5s"

# set maxmimum retry rate (percent) or minimum data rate (Mbps)
# roam is entered when
retry_rate = 50
data_rate = 20 #Mbps
# modifier to penalize score of unhealthy AP, needed to encourage roaming to different AP
unhealthy_score_mod = 20

# These values define upper and lower bounds RSSI must cross before re-roaming
# This is meant to prevent ping-ponging when at RSSI boundry
rssi_hysteresis_up = 5
rssi_hysteresis_down = 5

# This value is used as an upward RSSI buffer when the roaming tier is evaluated
# It prevents frequent tier oscilation when at an rssi boundry
tier_hysteresis = 5

# Set number of samples to avg rssi over
# Alleviates transient RSSI changes triggering roam logic
rssi_smooth_window = 5

[timing]
# Timings for signal polling (e.g. hosts current RSSI, SNR) and bg scan interval
sig_poll_interval = "250ms"
base_scan_interval = "30s"

# Defines number of bssids used to build fast-scan channel list
# In very dense environments, this number can be tuned to optimize channel scanning
max_bss_ct = 15

[score_weights]
# These tune the scoring algorithm, which ranks candidate APs
# Set to 100 for maxmium weight, 0 to ignore category
rssi = 100
snr = 0
qbss_util = 15
band = 35
channel_width = 10
phy_type = 15

# RSSI has a multiplicative below the knee, using an exponential curve
rssi_knee = -68
rssi_exponent = 1.8

[score_clamps]
# Used for RSSI and SNR scoring.
# Values below min are scored 0, values above max are score 100
# Values between clamps are scored linearly
min_rssi = -82
max_rssi = -25
min_snr = 10
max_snr = 50

# Use the following to adjust various scoring. This is where you can 
# tweak band pref, cw pref, etc.
[band_scores]
2point4ghz = 20
5ghz = 80
6ghz = 100

[chan_width_scores]
20mhz = 40
40mhz = 40
80mhz = 75
160mhz = 75
320mhz = 75

[phy_scores]
legacy = 0
80211n = 20
80211ac = 50
80211ax = 80
80211be = 100
`

const macOSTemplate = `
# roamctl macOS profile
# Approximates documented Apple macOS roaming behavior
# Reference: https://support.apple.com/guide/deployment/wi-fi-roaming-support-dep98f116c0f/web

[preferences]
interface = "wlan0"

[roaming_tiers]
# These values set the floor of each RSSI tier, which dictate roaming logic
# i.e. if fair_rssi is -67, -68 is in the "degraded" tier.
excellent_rssi = -58
# Apple macOS roam trigger threshold is -75 dBm — degraded tier floor set to match
fair_rssi = -68
degraded_rssi = -75
# Values lower than degraded are considered critical

# Apple macOS requires candidate AP to be 12 dB stronger regardless of activity
# score_delta approximates this relative signal strength requirement
fair_score_delta = 10
degraded_score_delta = 12
critical_score_delta = 6

[stability]
# sets cooldown duration after connection changed
# prevents roaming while cooldown is in effect
connection_cooldown = "5s"

# Set number of samples to avg rssi over
# Alleviates transient RSSI changes triggering roam logic
rssi_smooth_window = 5

# 12 dB stronger requirement from Apple effectively acts as hysteresis
# These values approximate that behavior at the tier boundary
rssi_hysteresis_up = 5
rssi_hysteresis_down = 5
tier_hysteresis = 5

# set maxmimum retry rate (percent) or minimum data rate (Mbps)
# roam is entered when
retry_rate = 50
data_rate = 20 #Mbps
# modifier to penalize score of unhealthy AP, needed to encourage roaming to different AP
unhealthy_score_mod = 20


[timing]
sig_poll_interval = "250ms"
base_scan_interval = "15s"
max_bss_ct = 15

[score_weights]
rssi = 100
snr = 0
qbss_util = 30
# Apple prefers 5/6 GHz — band weight reflects this
band = 50
# Apple explicitly prefers wider channel widths
channel_width = 10
# Apple prefers newer PHY generations (Wi-Fi 7 > 6 > 5 > 4)
phy_type = 15

# RSSI has a multiplicative below the knee, using an exponential curve
rssi_knee = -67
rssi_exponent = 2.5

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

const iOSTemplate = `
# roamctl iOS/iPadOS profile
# Approximates documented Apple iOS/iPadOS roaming behavior
# Reference: https://support.apple.com/guide/deployment/wi-fi-roaming-support-dep98f116c0f/web

[preferences]
interface = "wlan0"

[roaming_tiers]
# These values set the floor of each RSSI tier, which dictate roaming logic
# i.e. if fair_rssi is -67, -68 is in the "degraded" tier.
excellent_rssi = -58
# Apple iOS/iPadOS roam trigger threshold is -70 dBm — degraded tier floor set to match
fair_rssi = -63
degraded_rssi = -70
# Values lower than degraded are considered critical

# Apple iOS requires candidate to be 8 dB stronger when active, 12 dB when idle
# score_delta approximates the active (more permissive) threshold
fair_score_delta = 8
degraded_score_delta = 8
critical_score_delta = 4

[stability]
# sets cooldown duration after connection changed
# prevents roaming while cooldown is in effect
connection_cooldown = "5s"

# iOS roams more aggressively than macOS — tighter hysteresis reflects this
rssi_hysteresis_up = 4
rssi_hysteresis_down = 4
tier_hysteresis = 4

# set maxmimum retry rate (percent) or minimum data rate (Mbps)
# roam is entered when
retry_rate = 75
data_rate = 10 #Mbps
# modifier to penalize score of unhealthy AP, needed to encourage roaming to different AP
unhealthy_score_mod = 20

# Set number of samples to avg rssi over
# Alleviates transient RSSI changes triggering roam logic
rssi_smooth_window = 5


[timing]
sig_poll_interval = "250ms"
base_scan_interval = "15s"
max_bss_ct = 15

[score_weights]
rssi = 100
snr = 0
qbss_util = 30
# Apple prefers 5/6 GHz — band weight reflects this
band = 50
# Apple explicitly prefers wider channel widths
channel_width = 10
# Apple prefers newer PHY generations (Wi-Fi 7 > 6 > 5 > 4)
phy_type = 15

# RSSI has a multiplicative below the knee, using an exponential curve
rssi_knee = -67
rssi_exponent = 2.5

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
