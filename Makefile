.PHONY: test validate install run deploy

PI ?= pi@raspberrypi.local

test:
	python3 -m unittest discover -s ampel/tests -t .

validate:
	python3 -m ampel.main -config config.toml -validate

install: test
	sudo sh deploy/install.sh .

run:
	python3 -m ampel.main -config config.toml

deploy:
	ssh $(PI) 'cd ampel && git pull && sudo sh deploy/install.sh . && sudo systemctl restart ampel'
