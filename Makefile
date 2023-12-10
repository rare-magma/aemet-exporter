.PHONY: install
install:
	@mkdir --parents $${HOME}/.local/bin \
	&& mkdir --parents $${HOME}/.config/systemd/user \
	&& cp aemet_exporter.sh $${HOME}/.local/bin/ \
	&& chmod +x $${HOME}/.local/bin/aemet_exporter.sh \
	&& cp --no-clobber aemet_exporter.conf $${HOME}/.config/aemet_exporter.conf \
	&& chmod 400 $${HOME}/.config/aemet_exporter.conf \
	&& cp aemet-exporter.timer $${HOME}/.config/systemd/user/ \
	&& cp aemet-exporter.service $${HOME}/.config/systemd/user/ \
	&& systemctl --user enable --now aemet-exporter.timer

.PHONY: uninstall
uninstall:
	@rm -f $${HOME}/.local/bin/aemet_exporter.sh \
	&& rm -f $${HOME}/.config/aemet_exporter.conf \
	&& systemctl --user disable --now aemet-exporter.timer \
	&& rm -f $${HOME}/.config/.config/systemd/user/aemet-exporter.timer \
	&& rm -f $${HOME}/.config/systemd/user/aemet-exporter.service
