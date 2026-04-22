package config

const defaultConfigTemplate = `
[preferences]
# Toggle support for 802.11v (bss transition mgmt)
enable_btm = true

[roaming_tiers]
# These values set the floor of each RSSI tier, which dictate roaming logic
# i.e. if fair_rssi is -67, -68 is in the "degraded" tier.
excellent_rssi = -50
fair_rssi = -65
degraded_rssi = -73
# values lower than degraded are considered critical

# Set score deltas required to roam per tier
# Higher numbers mean candidate AP must be significantly better
fair_score_delta = 7
degraded_score_delta = 6
critical_score_delta = 4

[stability]
# sets cooldown duration after connection changed
# prevents roaming while cooldown is in effect
connection_cooldown = "5s"

# set maxmimum retry rate (percent) or minimum data rate (Mbps) or min MCS index
# Connection considered unstable when values outside of these bounds
# Unstable connection causes a roam attempt to fire
retry_rate = 75
data_rate = 10 #Mbps
mcs_index = 2
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
# Must be an integer from 1 to 10. 0 is not allowed.
rssi_smooth_window = 5

[timing]
# Timings for signal polling (e.g. hosts current RSSI, SNR) and bg scan interval
sig_poll_interval = "100ms"
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
2point4ghz = 0
5ghz = 80
6ghz = 100

[chan_width_scores]
20mhz = 40
40mhz = 40
80mhz = 75
160mhz = 85
320mhz = 100

[phy_scores]
legacy = 0
80211n = 20
80211ac = 50
80211ax = 80
80211be = 100
`
