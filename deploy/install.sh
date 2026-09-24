#!/bin/sh
# Legt Nutzer, Verzeichnisse und Dienst an. Aufruf auf dem Pi als root:
#   sudo sh install.sh [quellverzeichnis]
set -eu

SRC=${1:-$(dirname "$0")}
if [ -f "$SRC/configs/config.yaml" ]; then
	REPO=$SRC
	SRC=$SRC/deploy
	[ -f "$REPO/bin/ampel" ] && BINARY=$REPO/bin/ampel
	[ -f "$REPO/bin/ampelctl" ] && CTL=$REPO/bin/ampelctl
	CONFIGSRC=$REPO/configs/config.yaml
	DOCSRC=$REPO/docs
fi
BIN=/usr/local/bin/ampel
CONFIG=/etc/ampel/config.yaml
UNIT=/etc/systemd/system/ampel.service
DOCS=/usr/local/share/doc/ampel

if [ "$(id -u)" -ne 0 ]; then
	echo "Dieses Skript braucht root-Rechte." >&2
	exit 1
fi

binary=${BINARY:-$SRC/ampel-armv7}
[ -f "$binary" ] || binary=$SRC/ampel
if [ ! -f "$binary" ]; then
	echo "Kein Programm in $SRC gefunden, erwartet ampel-armv7 oder ampel." >&2
	exit 1
fi

# Die Gruppe gpio bringt Raspberry Pi OS mit. Fehlt sie, gehoert /dev/gpiochip0 niemandem,
# den der Dienst erreichen kann.
for group in gpio spi; do
	if ! getent group "$group" >/dev/null; then
		groupadd --system "$group"
	fi
done
if ! getent passwd ampel >/dev/null; then
	useradd --system --no-create-home --shell /usr/sbin/nologin --gid gpio ampel
fi
usermod --append --groups gpio,spi ampel

install -d -m 0755 /etc/ampel "$DOCS"
install -m 0755 "$binary" "$BIN"

ctl=${CTL:-$SRC/ampelctl-armv7}
[ -f "$ctl" ] || ctl=$SRC/ampelctl
if [ -f "$ctl" ]; then
	install -m 0755 "$ctl" /usr/local/bin/ampelctl
fi


if [ -f "$CONFIG" ]; then
	echo "$CONFIG bleibt unveraendert, neue Vorlage liegt als $CONFIG.neu"
	install -m 0644 "${CONFIGSRC:-$SRC/config.yaml}" "$CONFIG.neu"
else
	install -m 0644 "${CONFIGSRC:-$SRC/config.yaml}" "$CONFIG"
fi
install -m 0644 "$SRC/ampel.service" "$UNIT"
for doc in aufbau.md vorfuehrung.md architektur.md projektziel.md; do
	src=${DOCSRC:-$SRC}/$doc
	if [ -f "$src" ]; then
		install -m 0644 "$src" "$DOCS/$doc"
	fi
done

# Die Anzeige haengt an SPI. Ohne diese Zeile in /boot/config.txt gibt es kein spidev.
if [ -e /boot/config.txt ] && ! grep -q "^dtparam=spi=on" /boot/config.txt; then
	echo "dtparam=spi=on" >> /boot/config.txt
	echo "SPI eingeschaltet, die Anzeige arbeitet erst nach einem Neustart."
fi

"$BIN" -config "$CONFIG" -validate

systemctl daemon-reload
systemctl enable --now ampel.service
systemctl --no-pager --lines=5 status ampel.service
