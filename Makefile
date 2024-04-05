#!/bin/sh

PROGNAME=mfetch

# build targets
$(PROGNAME): *.go
	@env GOPATH=/tmp/go go get -d && env GOPATH=/tmp/go CGO_ENABLED=0 GOARCH=${_ARCH} GOOS=${_OS} go build -trimpath -o $(PROGNAME)
	@-strip $(PROGNAME) 2>/dev/null || true
	@-#upx -9 $(PROGNAME) 2>/dev/null || true
clean:
distclean:
	@rm -f $(PROGNAME) *.upx

# run targets
server: $(PROGNAME)
	@./$(PROGNAME) -verbose -listen :8000
client: $(PROGNAME)
	@./$(PROGNAME) -verbose -dump http://localhost:8000/100G
