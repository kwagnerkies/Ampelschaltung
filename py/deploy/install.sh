#!/bin/sh
set -eu

SRC=${1:-$(dirname "$0")/..}
LIB=/usr/local/lib/ampel
CONFIG=/etc/ampel/config.toml

if [ "$(id -u)" -ne 0 ]; then
	echo "Dieses Skript braucht root-Rechte." >&2
	exit 1
fi

apt-get install -y python3-gpiozero >/dev/null 2>&1 || true

for group in gpio spi; do
	getent group "$group" >/dev/null || groupadd --system "$group"
done
getent passwd ampel >/dev/null || useradd --system --no-create-home --shell /usr/sbin/nologin --gid gpio ampel
usermod --append --groups gpio,spi ampel

install -d -m 0755 /etc/ampel "$LIB/ampel"
install -m 0644 "$SRC"/ampel/*.py "$LIB/ampel/"
install -m 0755 "$SRC/ampelctl" /usr/local/bin/ampelctl
install -m 0644 "$SRC/deploy/ampel.service" /etc/systemd/system/ampel.service
[ -f "$CONFIG" ] || install -m 0644 "$SRC/config.toml" "$CONFIG"

if [ -e /boot/config.txt ]; then
	for line in "dtparam=spi=on" "dtoverlay=spi1-1cs"; do
		grep -q "^$line" /boot/config.txt || {
			echo "$line" >> /boot/config.txt
			echo "$line eingetragen, wirksam nach einem Neustart."
		}
	done
fi

PYTHONPATH=$LIB python3 -m ampel.main -config "$CONFIG" -validate
systemctl daemon-reload
systemctl enable --now ampel.service
systemctl --no-pager --lines=5 status ampel.service
