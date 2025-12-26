package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pyke369/golang-support/auth"
	"github.com/pyke369/golang-support/dynacert"
	l "github.com/pyke369/golang-support/listener"
	"github.com/pyke369/golang-support/rcache"
	"github.com/pyke369/golang-support/ustr"
)

var (
	serverPayload  = make([]byte, 64<<10)
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
			response.Header().Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(size, 10))
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
		response.Header().Set("Server", PROGNAME+"/"+PROGVER)
		response.Header().Set("Access-Control-Allow-Origin", "*")
		response.Header().Set("Access-Control-Allow-Methods", "OPTIONS, HEAD, GET")
		response.Header().Set("Access-Control-Allow-Headers", "Range")
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
			if match, _ := auth.Password(received, []string{Password}, false); !match {
				response.Header().Set("WWW-Authenticate", `Basic realm="`+PROGNAME+`"`)
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
			serverMessages <- strings.Join([]string{
				"S",
				ustr.Int(int(id%10000), 4, 1),
				start.Format("15:04:05.000"),
				request.RemoteAddr,
				request.Method,
				request.URL.Path,
				srange,
			}, "|")
		}
		handler.ServeHTTP(&writer, request)
		elapsed := time.Since(start)
		switch {
		case elapsed >= time.Second:
			elapsed = elapsed.Truncate(10 * time.Millisecond)

		case elapsed < time.Millisecond:
			elapsed = elapsed.Truncate(time.Microsecond)

		default:
			elapsed = elapsed.Truncate(time.Millisecond)
		}
		if Dump {
			rrange := "-"
			if captures := rcache.Get(`^bytes (\d+-\d+/\d+)$`).FindStringSubmatch(writer.Header().Get("Content-Range")); captures != nil {
				rrange = captures[1]
			}
			serverMessages <- strings.Join([]string{
				"E",
				ustr.Int(int(id%10000), 4, 1),
				time.Now().Format("15:04:05.000"),
				request.RemoteAddr,
				request.Method,
				request.URL.Path,
				srange,
				strconv.Itoa(writer.status),
				strconv.FormatInt(writer.sent, 10),
				rrange,
				elapsed.String(),
				utilBandwidth((float64(writer.sent) * 8) / (float64(elapsed) / float64(time.Second))),
			}, "|")
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
				os.Stderr.WriteString("\r" + message + "     \n")

			case <-time.After(time.Second):
			}
			current := atomic.LoadInt64(&serverSent)
			if previous == 0 {
				previous = current
			}
			if elapsed := time.Since(start); elapsed >= 100*time.Millisecond && Verbose {
				os.Stderr.WriteString("\r" + strconv.FormatInt(atomic.LoadInt64(&serverInflight), 10) + " | " +
					utilBandwidth((float64(current-previous)*8)/(float64(elapsed)/float64(time.Second))) + "     ")
				previous = current
			}
		}
	}()

	mux, handler := http.NewServeMux(), serverSimulate()
	if Flagset.NArg() > 0 {
		handler = http.FileServer(http.Dir(Flagset.Args()[0]))
	}
	mux.Handle("/", base(handler))

	server := &http.Server{
		Handler:     mux,
		ErrorLog:    log.New(io.Discard, "", 0),
		IdleTimeout: time.Duration(Timeout) * time.Second * 2,
		ReadTimeout: time.Duration(Timeout) * time.Second,
	}
	for {
		if listener, err := l.NewTCPListener("tcp", Listen, &l.TCPOptions{ReusePort: true}); err == nil {
			if Certificate != "" {
				certificate := &dynacert.DYNACERT{}
				if Certificate == "internal" {
					if key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err == nil {
						if der, err := x509.MarshalECPrivateKey(key); err == nil {
							pkey := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
							template := x509.Certificate{
								Subject:     pkix.Name{Organization: []string{PROGNAME}, CommonName: PROGNAME},
								NotBefore:   time.Now(),
								NotAfter:    time.Now().Add(10 * 365 * 24 * time.Hour),
								KeyUsage:    x509.KeyUsageDigitalSignature,
								ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
								DNSNames:    []string{PROGNAME},
							}
							if der, err := x509.CreateCertificate(rand.Reader, &template, &template, key.Public(), key); err == nil {
								certificate.Inline("*", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pkey)
							}
						}
					}

				} else if parts := strings.Split(Certificate, ","); len(parts) >= 2 {
					certificate.Add("*", parts[0], parts[1])
				}
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
