package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
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
	serverCertificate = [2]string{`-----BEGIN CERTIFICATE-----
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
	serverPayload  = bytes.Repeat([]byte{0}, 64<<10)
	serverId       = int64(0)
	serverInflight = int64(0)
	serverSent     = int64(0)
	serverMessages = make(chan string, 1<<10)
)

type serverWriter struct {
	rw     http.ResponseWriter
	status int
	sent   int64
}

func (sw *serverWriter) Header() http.Header {
	return sw.rw.Header()
}
func (sw *serverWriter) WriteHeader(status int) {
	sw.status = status
	sw.rw.WriteHeader(status)
}
func (sw *serverWriter) Write(data []byte) (n int, err error) {
	sw.sent += int64(len(data))
	atomic.AddInt64(&serverSent, int64(len(data)))
	return sw.rw.Write(data)
}

func serverSimulate() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Accept-Ranges", "bytes")
		captures := rcache.Get(`^/(\d+)([kmg]i?b?)?$`).FindStringSubmatch(strings.ToLower(request.URL.Path))
		if captures == nil {
			response.WriteHeader(http.StatusNotFound)
			return
		}
		size, _ := strconv.ParseInt(captures[1], 10, 64)
		base := int64(1000)
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

		captures, start, end := rcache.Get(`^bytes=(\d+)-(\d*)$`).FindStringSubmatch(request.Header.Get("Range")), int64(0), size-1
		if captures != nil {
			start, _ = strconv.ParseInt(captures[1], 10, 64)
			if start >= size {
				response.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			if value, err := strconv.ParseInt(captures[2], 10, 64); err == nil {
				end = value
				if end < start {
					response.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
					return
				}
				end = min(size-1, end)
			}
			response.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
			size = end - start + 1
		}

		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		if captures != nil {
			response.WriteHeader(http.StatusPartialContent)
		}
		if request.Method == http.MethodHead {
			return
		}
		for size > 0 {
			sent, err := response.Write(serverPayload[:min(len(serverPayload), int(size))])
			if err != nil {
				break
			}
			size -= int64(sent)
		}
	})
}

func base(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Server", fmt.Sprintf("%s/%s", PROGNAME, PROGVER))
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
			if match, _ := acl.Password(received, []string{Password}, false); !match {
				response.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, PROGNAME))
				response.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		if request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/.") || strings.Contains(request.URL.Path[1:], "/") {
			response.WriteHeader(http.StatusNotFound)
			return
		}

		atomic.AddInt64(&serverInflight, 1)
		id, start, writer, srange := atomic.AddInt64(&serverId, 1), time.Now(), serverWriter{rw: response, status: 200}, "-"
		if captures := rcache.Get(`^bytes=(\d+)-(\d*)$`).FindStringSubmatch(request.Header.Get("Range")); captures != nil {
			srange = captures[1] + "-" + captures[2]
		}
		if Dump {
			serverMessages <- fmt.Sprintf("S|%04d|%s|%s|%s|%s", id%10000, start.Format("15:04:05.000"),
				request.RemoteAddr, request.URL.Path, srange)
		}
		handler.ServeHTTP(&writer, request)
		elapsed := time.Since(start)
		if elapsed >= time.Second {
			elapsed = elapsed.Truncate(10 * time.Millisecond)
		} else if elapsed < time.Millisecond {
			elapsed = elapsed.Truncate(time.Microsecond)
		} else {
			elapsed = elapsed.Truncate(time.Millisecond)
		}
		if Dump {
			serverMessages <- fmt.Sprintf("E|%04d|%s|%s|%s|%s|%d|%d|%v|%s", id%10000, time.Now().Format("15:04:05.000"),
				request.RemoteAddr, request.URL.Path, srange, writer.status, writer.sent, elapsed, utilBandwidth((float64(writer.sent)*8)/(float64(elapsed)/float64(time.Second))))
		}
		atomic.AddInt64(&serverInflight, -1)
	})
}

func Server() {
	go func() {
		previous := int64(0)
		for {
			start := time.Now()
			select {
			case message := <-serverMessages:
				fmt.Fprintf(os.Stderr, "\r%s     \n", message)
			case <-time.After(time.Second):
			}
			current := atomic.LoadInt64(&serverSent)
			if previous == 0 {
				previous = current
			}
			if elapsed := time.Since(start); elapsed >= 100*time.Millisecond && Verbose {
				fmt.Fprintf(os.Stderr, "\r%d | %s     ", atomic.LoadInt64(&serverInflight), utilBandwidth((float64(current-previous)*8)/(float64(elapsed)/float64(time.Second))))
				previous = current
			}
		}
	}()

	cpaths := []string{}
	if Certificate == "internal" {
		cpaths = []string{fmt.Sprintf("%s/%s-cert-%d.pem", os.TempDir(), PROGNAME, os.Getpid()), fmt.Sprintf("%s/%s-key-%d.pem", os.TempDir(), PROGNAME, os.Getpid())}
		ioutil.WriteFile(cpaths[0], []byte(serverCertificate[0]), 0644)
		ioutil.WriteFile(cpaths[1], []byte(serverCertificate[1]), 0600)
		go func() {
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
			<-signals
			for _, path := range cpaths {
				os.Remove(path)
			}
			os.Exit(0)
		}()
	} else if value := strings.SplitN(Certificate, ",", 2); len(value) == 2 {
		cpaths = []string{strings.TrimSpace(value[0]), strings.TrimSpace(value[1])}
	}

	mux, handler := http.NewServeMux(), serverSimulate()
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
			if len(cpaths) == 2 {
				certificate := &dynacert.DYNACERT{}
				certificate.Add("*", cpaths[0], cpaths[1])
				server.TLSConfig = certificate.TLSConfig()
				server.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
				server.ServeTLS(listener, "", "")
			} else {
				server.Serve(listener)
			}
		}
		time.Sleep(time.Second)
	}
}
