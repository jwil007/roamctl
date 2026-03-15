This project uses wpa_supplicant gymnastics (via the Unix Socket ctrl interface https://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html) to take over the autonomous roaming algorithm and let you set your own roaming thresholds, like RSSI.

Right now just RSSI but it'll be better soon.......

To use, clone the repo, install Golang, and run go build. Then run ./roamctl -i *interface name* -r *RSSI cutoff* Default vals are -i: wlan0 and -r: -65
