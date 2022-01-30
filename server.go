package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pyke369/golang-support/acl"
	"github.com/pyke369/golang-support/dynacert"
	"github.com/pyke369/golang-support/listener"
	"github.com/pyke369/golang-support/rcache"
)

// internal self-signed TLS certificate
// Serial:     58:c4:a8:c1:27:45:86:c0:e1:52:16:ab:d3:e4:3d:72:fb:b5:86:09
// Issuer:     CN = mfetch
// Subject:    CN = mfetch
// Not Before: Jan 21 15:04:26 2021 GMT
// Not After : Jan 19 15:04:26 2031 GMT
var (
	internal = [2]string{`-----BEGIN CERTIFICATE-----
MIIDAzCCAeugAwIBAgIUWMSowSdFhsDhUhar0+Q9cvu1hgkwDQYJKoZIhvcNAQEL
BQAwETEPMA0GA1UEAwwGbWZldGNoMB4XDTIxMDEyMTE1MDQyNloXDTMxMDExOTE1
MDQyNlowETEPMA0GA1UEAwwGbWZldGNoMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEA5fhnLwnmclzmHr01pobLlJ35UJkj1NJ3jWUVNkTnlITEBlkk9/Ta
35+VU8bLIBxgE9oKQyN0E5YGiT9D8uBp2jn7HAu8vDUFh1Ao8XEzdSgFjPtvPmH1
fSQ9b4qomDUDIebX8WDOox4VEwPcgz0Q0DF6cTmMs99kuyuhEyehdDRgSN/b7Qnf
FDAEJzXh2xZXTMjngPRF2cAHgGD6rh137JjUHoJ5kZ8iPxOp2cjaaw51HxafuFbN
aLb86n7g84hj51I8TzQjEM66rB1QvwsJVgBvSnX8ZkI+v4K5Y4ttzfQU2DD6wd9N
lIcdVPPmAtn5wPjMJyuC1HIAx8dNe/F+wwIDAQABo1MwUTAdBgNVHQ4EFgQUrj8O
XiNQfr1SSwtd6DlC/3Irb10wHwYDVR0jBBgwFoAUrj8OXiNQfr1SSwtd6DlC/3Ir
b10wDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAQf1urVUoiQqT
6t0OZfeqYOLqahR+0l7Sm7TsDwM8u86WHpwXvbK4ULVX2FFtz0LI0bcVLbK4OxX/
4vNYMf3Q8ssjEKVyOKa3yy75/b7z5ahjMBqfWTETnSschJE+tuG5Nl4oGYwEYZXP
r6Ay1QTVsKlKLTG0+yiHbNPBVxNbXMSZEd4YBeVqikNmvy2WgVs6FhcWFMlrM5XO
eEFr2Z6t93jLzyYKyjxxomuijOCYy/oYNvbLisnbBb2OqwsjgsrMb5q324zXdBoD
7Ksvi+Ns0emN358FA210ORTxMMG5MSBY8OB7hphtxxi3slMzMsQ4FBEl5kKj9eFl
7DuHwqRfCA==
-----END CERTIFICATE-----
`,
		`-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDl+GcvCeZyXOYe
vTWmhsuUnflQmSPU0neNZRU2ROeUhMQGWST39Nrfn5VTxssgHGAT2gpDI3QTlgaJ
P0Py4GnaOfscC7y8NQWHUCjxcTN1KAWM+28+YfV9JD1viqiYNQMh5tfxYM6jHhUT
A9yDPRDQMXpxOYyz32S7K6ETJ6F0NGBI39vtCd8UMAQnNeHbFldMyOeA9EXZwAeA
YPquHXfsmNQegnmRnyI/E6nZyNprDnUfFp+4Vs1otvzqfuDziGPnUjxPNCMQzrqs
HVC/CwlWAG9KdfxmQj6/grlji23N9BTYMPrB302Uhx1U8+YC2fnA+MwnK4LUcgDH
x0178X7DAgMBAAECggEAY0GJV3YQbn/GGrJTe6JmL6jXOIBARNTqIK7mLtwij6mV
6Z+EIzkdVrNMAjKk7SESHr9W+o9MxD9WZtpVe3h8d2HbDcnLFfhUgIiKg1r2eLRj
YOwMoYIqMG75zTCtf7Qxu+okfdvok+Kh+ekKveIXZaRVUpUiM2hR068LAHd0afB3
wE40boQ8kiRE31LLEqOV+Pzc/UWli7kH4ayhQLGhAIVeezGj+8E8FB4bU2GcvlEj
VZcB8Cxm42MSw9ucx/h+9DpG6RakgaBBD4lT9yxWt9oYOWX6AbvDSzn3jV/ueEct
Q5ho9WNqWvPXkOjbSk2aBge0cy5qipxY7f6quzN48QKBgQD1QOQ3Rbx3qZOtSSjC
fdT2RFrUCEQbMt7nXVgODQTqXBMTsaw7KjgurmCCroZ1pG0eI/g8eN06tktza8w8
tgiQq6GqrR0SBdZvaaIuVtm4yXetLLtSlhSJji/fIqnuqx/V2ygh2+2rNDEyo/aZ
708xRMltF7lwQiZ4YOUDHbsv/wKBgQDwDBL8kA4Kjkei6aOJbBLx438p18QbN7Lq
7qDaso4jgqwGnXRhSiiF7zPa3EuIj9cQ8TbmhcaCki7OtDyVChA1Ad1SaafgedSh
DzvpqAgQV/ZJIoGGP2kD0E9VfWWGhPBTF2kQ9VOB/ZgEK6epDL8JivUw4IncCt/t
dKmOUdjxPQKBgQDWRY2eJNVWjtexLBvqYNmxF2NroJUwVi+dYFZQYFuNDki0iiR5
xJc1YbB8PFLJcZDMJoz4+HgAlcgx3VqhKEEvdGRYo8qkNml1CYtihQrPgWWH7W7z
5p+m1o1InBZvqR61TzYu7uElFQJuxgXr08MSvpBlObcQNxs5TR6IrG8grQKBgQDg
UMglT5BveMmkiWQS9PU3KPoZ5dESBhihxWB3PcfpkyCiBd1NVPlNP1xbtuS2toOp
B1/gRz5bobMv4emC9KZ0gkuJycXg1LhH0W6RSD5Q14IEkcQr6XF+6NhZ8RZAgFX7
r7K08CubG5lEvG6uYITcrAe4JvtsrpTW1t/jaMSrmQKBgQCRcJDT1gJKnKpDKNKS
IaHIWZtLylFojEbmGrIS/5Xd8ez8EROhW5n+SZcenxT+2yVjJwhZAoELIXoPg09l
MJYHjCl6QyJlC4jv/NkchyPwYY11YubYXvxAjNnTB11uFLz2XpQF3+t5z60MBye/
GooiNcKqSQG5n1ivW1PdzY3T0Q==
-----END PRIVATE KEY-----
`}
	content = bytes.Repeat([]byte{0}, 1<<20)
)

func simulate() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Accept-Ranges", "bytes")
		captures := rcache.Get(`^/(\d+)([kmg]i?b?)?$`).FindStringSubmatch(strings.ToLower(request.URL.Path))
		if captures == nil {
			response.WriteHeader(http.StatusNotFound)
			return
		}

		size, _ := strconv.Atoi(captures[1])
		base := 1000
		if strings.Contains(captures[2], "i") {
			base = 1024
		}
		if strings.HasPrefix(captures[2], "k") {
			size *= base
		}
		if strings.HasPrefix(captures[2], "m") {
			size *= base * base
		}
		if strings.HasPrefix(captures[2], "g") {
			size *= base * base * base
		}
		if size <= 0 {
			response.WriteHeader(http.StatusNotFound)
			return
		}

		start, end, captures := 0, size-1, rcache.Get(`^bytes=(\d+)-(\d*)$`).FindStringSubmatch(request.Header.Get("Range"))
		if captures != nil {
			start, _ = strconv.Atoi(captures[1])
			if start >= size {
				response.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			if value, err := strconv.Atoi(captures[2]); err == nil {
				end = value
				if end < start {
					response.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
					return
				}
				end = int(math.Min(float64(size-1), float64(end)))
			}
			response.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
			size = end - start + 1
		}

		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("Content-Length", fmt.Sprintf("%d", size))
		if captures != nil {
			response.WriteHeader(http.StatusPartialContent)
		}
		if request.Method == http.MethodHead {
			return
		}
		for size > 0 {
			sent, err := response.Write(content[:int(math.Min(float64(len(content)), float64(size)))])
			if err != nil {
				break
			}
			size -= sent
		}
	})
}

func base(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Server", fmt.Sprintf("%s/%s", PROGNAME, VERSION))
		response.Header().Set("Access-Control-Allow-Origin", "*")
		response.Header().Set("Access-Control-Allow-Methods", "OPTIONS, HEAD, GET")
		response.Header().Set("Access-Control-Max-Age", "86400")
		if request.Method == http.MethodOptions {
			return
		}
		if request.Method != http.MethodHead && request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if Password != "" {
			_, received, _ := request.BasicAuth()
			if !acl.Password(received, []string{Password}) {
				response.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, PROGNAME))
				response.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		if request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/.") || strings.Contains(request.URL.Path[1:], "/") {
			response.WriteHeader(http.StatusNotFound)
			return
		}
		handler.ServeHTTP(response, request)
	})
}

func Server() {
	tlspaths := []string{}
	if Certificate == "internal" {
		tlspaths = []string{fmt.Sprintf("%s/%s-cert-%d.pem", os.TempDir(), PROGNAME, os.Getpid()), fmt.Sprintf("%s/%s-key-%d.pem", os.TempDir(), PROGNAME, os.Getpid())}
		ioutil.WriteFile(tlspaths[0], []byte(internal[0]), 0644)
		ioutil.WriteFile(tlspaths[1], []byte(internal[1]), 0600)
		go func() {
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
			<-signals
			for _, path := range tlspaths {
				os.Remove(path)
			}
			os.Exit(0)
		}()
	} else if value := strings.SplitN(Certificate, ",", 2); len(value) == 2 {
		tlspaths = []string{strings.TrimSpace(value[0]), strings.TrimSpace(value[1])}
	}

	mux, handler := http.NewServeMux(), simulate()
	if Flagset.NArg() > 0 {
		handler = http.FileServer(http.Dir(Flagset.Args()[0]))
	}
	mux.Handle("/", base(handler))

	server := &http.Server{
		Handler:     mux,
		ErrorLog:    log.New(ioutil.Discard, "", 0),
		IdleTimeout: time.Duration(Timeout) * time.Second * 2,
		ReadTimeout: time.Duration(Timeout) * time.Second,
	}
	for {
		if listener, err := listener.NewTCPListener("tcp", Listen, true, 0, 0, nil); err == nil {
			if len(tlspaths) == 2 {
				certificates := &dynacert.DYNACERT{}
				certificates.Add("*", tlspaths[0], tlspaths[1])
				server.TLSConfig = dynacert.IntermediateTLSConfig(certificates.GetCertificate)
				server.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
				server.ServeTLS(listener, "", "")
			} else {
				server.Serve(listener)
			}
		}
		time.Sleep(time.Second)
	}
}
