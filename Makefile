.PHONY: install
install:
	@mkdir --parents $${HOME}/.local/bin \
	&& mkdir --parents $${HOME}/.config/systemd/user \
	&& cp aemet_exporter $${HOME}/.local/bin/ \
	&& chmod +x $${HOME}/.local/bin/aemet_exporter \
	&& cp --no-clobber aemet_exporter.json $${HOME}/.config/aemet_exporter.json \
	&& chmod 400 $${HOME}/.config/aemet_exporter.json \
	&& cp aemet-exporter.timer $${HOME}/.config/systemd/user/ \
	&& cp aemet-exporter.service $${HOME}/.config/systemd/user/ \
	&& systemctl --user enable --now aemet-exporter.timer

.PHONY: uninstall
uninstall:
	@rm -f $${HOME}/.local/bin/aemet_exporter \
	&& rm -f $${HOME}/.config/aemet_exporter.json \
	&& systemctl --user disable --now aemet-exporter.timer \
	&& rm -f $${HOME}/.config/.config/systemd/user/aemet-exporter.timer \
	&& rm -f $${HOME}/.config/systemd/user/aemet-exporter.service

.PHONY: build
build:
	@go build -ldflags="-s -w" -o aemet_exporter main.go