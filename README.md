# ha-ghostcontrols-mqtt
Simple Go program for controlling a GhostControls estate gate via MQTT on a Raspberry Pi and a 433Mhz antenna.

## Install

Build, drop the binary and the unit in place, then enable the service:

```sh
go build -o ha-ghostcontrols-mqtt .
sudo install -m 0755 ha-ghostcontrols-mqtt /usr/local/bin/
sudo install -m 0644 ha-ghostcontrols-mqtt.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now ha-ghostcontrols-mqtt.service
```

The service runs as root because `sendook` needs direct GPIO access to
drive the 433 MHz transmitter. Logs: `journalctl -u ha-ghostcontrols-mqtt -f`.
