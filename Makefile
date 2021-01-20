#!/bin/sh

# build targets
mfetch: *.go
	@env GOPATH=/tmp/go CGO_ENABLED=0 go build -trimpath -o mfetch
	@-strip mfetch 2>/dev/null || true
	@-upx -9 mfetch 2>/dev/null || true
clean:
distclean:
	@rm -f mfetch *.upx
deb:
	@debuild -e GOROOT -e GOPATH -e PATH -i -us -uc -b
debclean:
	@debuild -- clean
	@rm -f ../mfetch_*

# run targets
client: mfetch
	@./mfetch --verbose --trustpeer https://login:password@myserver.com/20GB 20GB
server: mfetch
	@./mfetch --listen :443 --tls internal --password password .
