#!/bin/sh
# Legt Nutzer, Verzeichnisse und Dienst an. Aufruf auf dem Pi als root:
#   sudo sh install.sh [quellverzeichnis]
set -eu

SRC=${1:-$(dirname "$0")}
BIN=/usr/local/bin/ampel
CONFIG=/etc/ampel/config.yaml
UNIT=/etc/systemd/system/ampel.service
DOCS=/usr/local/share/doc/ampel

if [ "$(id -u)" -ne 0 ]; then
	echo "Dieses Skript braucht root-Rechte." >&2
	exit 1
fi

binary=$SRC/ampel-armv7
[ -f "$binary" ] || binary=$SRC/ampel
if [ ! -f "$binary" ]; then
	echo "Kein Programm in $SRC gefunden, erwartet ampel-armv7 oder ampel." >&2
	exit 1
fi

# Die Gruppe gpio bringt Raspberry Pi OS mit. Fehlt sie, gehoert /dev/gpiochip0 niemandem,
# den der Dienst erreichen kann.
if ! getent group gpio >/dev/null; then
	groupadd --system gpio
fi
if ! getent passwd ampel >/dev/null; then
	useradd --system --no-create-home --shell /usr/sbin/nologin --gid gpio ampel
else
	usermod --append --groups gpio ampel
fi

install -d -m 0755 /etc/ampel "$DOCS"
install -d -m 0755 -o ampel -g gpio /var/log/ampel /var/lib/ampel
install -m 0755 "$binary" "$BIN"

if [ -f "$CONFIG" ]; then
	echo "$CONFIG bleibt unveraendert, neue Vorlage liegt als $CONFIG.neu"
	install -m 0644 "$SRC/config.yaml" "$CONFIG.neu"
else
	install -m 0644 "$SRC/config.yaml" "$CONFIG"
fi
install -m 0644 "$SRC/ampel.service" "$UNIT"
for doc in aufbau.md auswertung.md vorfuehrung.md; do
	[ -f "$SRC/$doc" ] && install -m 0644 "$SRC/$doc" "$DOCS/$doc"
done

"$BIN" -config "$CONFIG" -validate

systemctl daemon-reload
systemctl enable --now ampel.service
systemctl --no-pager --lines=5 status ampel.service
