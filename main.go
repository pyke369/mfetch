package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

type RESUME struct {
	Size        int64      `json:"size"`
	Modified    int64      `json:"modified"`
	Concurrency int        `json:"concurrency"`
	Chunks      [][3]int64 `json:"chunks"`
}

var (
	progname = "mfetch"
	version  = "1.0.2"
	progress = false
)

// server mode requests handler
func shandler(password string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Server", fmt.Sprintf("%s/%s", progname, version))
		if password != "" {
			if _, received, ok := request.BasicAuth(); !ok || received != password {
				response.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, progname))
				http.Error(response, "", http.StatusUnauthorized)
				return
			}
		}
		if request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/.") || strings.Contains(request.URL.Path[1:], "/") {
			http.Error(response, "", http.StatusNotFound)
			return
		}
		handler.ServeHTTP(response, request)
	})
}

// main program entry
func main() {

	// parse command-line arguments
	concurrency, headers, trustpeer, verbose, noresume, listen, tlspair, password := 10, multiflag{}, false, false, false, "", "", ""
	fset := flag.NewFlagSet("lambda-gateway", flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, `usage:
  %s [<options>] <argument(s)>

arguments:
  - client mode
        <remote-url> <local-file>
  - server mode
        <shared-folder>

options:
`, filepath.Base(os.Args[0]))
		fset.PrintDefaults()
	}
	fset.IntVar(&concurrency, "concurrency", concurrency, "transfer concurrency level")
	fset.Var(&headers, "header", "add arbitrary HTTP header to requests (repeatable)")
	fset.BoolVar(&trustpeer, "trustpeer", trustpeer, "ignore server TLS certificate errors (default false)")
	fset.BoolVar(&verbose, "verbose", verbose, "verbose mode (default false)")
	fset.BoolVar(&progress, "progress", progress, "emit transfer progress JSON indications (default false)")
	fset.BoolVar(&noresume, "noresume", noresume, "ignore transfer auto-resuming (default false)")
	fset.StringVar(&listen, "listen", listen, "listening address & port in server mode (default client mode)")
	fset.StringVar(&tlspair, "tls", tlspair, `TLS certificate & key to use in server mode (or "internal", default none)`)
	fset.StringVar(&password, "password", password, "security password in server mode (default none)")
	fset.Parse(os.Args[1:])
	listen, tlspair, password = strings.TrimLeft(strings.TrimSpace(listen), "*"), strings.TrimSpace(tlspair), strings.TrimSpace(password)
	if (listen != "" && fset.NArg() != 1) || (listen == "" && fset.NArg() != 2) {
		fset.Usage()
		os.Exit(1)
	}

	// graceful exit
	exit, cert, server := false, []string{}, &http.Server{}
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
		<-signals
		if tlspair == "internal" {
			for _, path := range cert {
				os.Remove(path)
			}
		}
		exit = true
		server.Shutdown(context.Background())
	}()

	// server mode
	if listen != "" {
		shared := fset.Args()[0]
		if tlspair != "" {
			if tlspair == "internal" {
				cert = []string{
					fmt.Sprintf("%s/_%s-cert-%d.pem", os.TempDir(), progname, os.Getpid()),
					fmt.Sprintf("%s/_%s-key-%d.pem", os.TempDir(), progname, os.Getpid()),
				}
				ioutil.WriteFile(cert[0], []byte(tlscert), 0644)
				ioutil.WriteFile(cert[1], []byte(tlskey), 0600)
			} else {
				if cert = strings.SplitN(tlspair, ",", 2); len(cert) != 2 {
					hfatal("invalid --tls option parameter", 2)
				}
				cert[0] = strings.TrimSpace(cert[0])
				cert[1] = strings.TrimSpace(cert[1])
			}
			server = &http.Server{
				Addr:         listen,
				Handler:      shandler(password, http.FileServer(http.Dir(shared))),
				TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
				TLSConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
					},
				},
			}
			if err := server.ListenAndServeTLS(cert[0], cert[1]); err != nil {
				hfatal(fmt.Sprintf("%v", err), 2)
			}
		} else {
			server = &http.Server{
				Addr:    listen,
				Handler: shandler(password, http.FileServer(http.Dir(shared))),
			}
			if err := server.ListenAndServe(); err != nil {
				hfatal(fmt.Sprintf("%v", err), 2)
			}
		}
		os.Exit(0)
	}

	// probe content (total document size + server byte-range requests support)
	remote, local := fset.Args()[0], fset.Args()[1]
	size, modified, ranges, transport := int64(0), int64(0), false, &http.Transport{ReadBufferSize: 64 << 10, TLSClientConfig: &tls.Config{}}
	if trustpeer {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if request, err := http.NewRequest(http.MethodHead, remote, nil); err == nil {
		for _, header := range headers {
			request.Header.Set(header[0], header[1])
		}
		request.Header.Set("User-Agent", fmt.Sprintf("%s/%s", progname, version))
		client := &http.Client{Timeout: 5 * time.Second, Transport: transport}
		if response, err := client.Do(request); err == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusOK {
				hfatal(fmt.Sprintf("HTTP status code %d", response.StatusCode), 2)
			}
			if response.Header.Get("Accept-Ranges") == "bytes" {
				ranges = true
			}
			if size = response.ContentLength; size <= 0 {
				hfatal(fmt.Sprintf("invalid document size %d", size), 2)
			}
			if last, err := http.ParseTime(response.Header.Get("Last-Modified")); err == nil {
				modified = last.Unix()
			}
		} else {
			hfatal(fmt.Sprintf("%v", err), 2)
		}
	} else {
		hfatal(fmt.Sprintf("%v", err), 2)
	}

	// adjust concurrency based on remote document size
	concurrency = int(math.Min(float64(size/(1<<20)), float64(concurrency)))
	concurrency = int(math.Min(100, math.Max(1, float64(concurrency))))
	if !ranges {
		concurrency = 1
	}

	// create or open destination file
	var handle *os.File

	resume, resumed, chunks, csize := "", false, [100][3]int64{}, size/int64(concurrency)
	for index := 0; index < concurrency; index++ {
		chunks[index][0], chunks[index][1] = csize*int64(index), (csize*(int64(index)+1))-1
		if index == concurrency-1 {
			chunks[index][1] = size - 1
		}
	}
	if local != "-" {
		resume = fmt.Sprintf("%s/._%s_resume", filepath.Dir(local), filepath.Base(local))
		if noresume || !ranges {
			os.Remove(resume)
			if !ranges {
				resume = ""
			}
		}
		if resume != "" {
			if content, err := ioutil.ReadFile(resume); err == nil {
				var state RESUME

				if json.Unmarshal(content, &state) == nil {
					if state.Size == size && state.Modified == modified {
						if info, err := os.Stat(local); err == nil && info.Size() == size && info.ModTime().Unix() >= modified {
							if handle, err = os.OpenFile(local, os.O_RDWR, 0644); err != nil {
								hfatal(fmt.Sprintf("%v", err), 3)
							}
							concurrency = state.Concurrency
							for index := 0; index < concurrency; index++ {
								chunks[index][0], chunks[index][1], chunks[index][2] = state.Chunks[index][0], state.Chunks[index][1], state.Chunks[index][2]
							}
							resumed = true
						}
					}
				}
			}
		}
		if !resumed {
			var err error

			if handle, err = os.OpenFile(local, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
				hfatal(fmt.Sprintf("%v", err), 3)
			}
			if err = handle.Truncate(size); err != nil {
				hfatal(fmt.Sprintf("%v", err), 3)
			}
		}
	}

	// start transfer workers
	sink := make(chan error, concurrency)
	for index := 0; index < concurrency; index++ {
		go func(index int) {
			start, end := chunks[index][0]+chunks[index][2], chunks[index][1]
			if start > end {
				return
			}
			if request, err := http.NewRequest(http.MethodGet, remote, nil); err == nil {
				for _, header := range headers {
					request.Header.Set(header[0], header[1])
				}
				request.Header.Set("User-Agent", fmt.Sprintf("%s/%s", progname, version))
				request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
				client := &http.Client{Transport: transport}
				if response, err := client.Do(request); err == nil {
					if response.StatusCode/100 != 2 {
						sink <- fmt.Errorf("invalid HTTP status code %d", response.StatusCode)
						return
					}
					if response.ContentLength != end-start+1 {
						sink <- fmt.Errorf("invalid content length (received %d instead of expected %d)", response.ContentLength, end-start+1)
						return
					}
					block := make([]byte, 64<<10)
					for start < end && !exit {
						read, err := response.Body.Read(block)
						if read > 0 {
							if handle != nil {
								if _, err := handle.WriteAt(block[:read], start); err != nil {
									sink <- err
									return
								}
							}
							atomic.AddInt64(&(chunks[index][2]), int64(read))
							start += int64(read)
						}
						if err != nil && read <= 0 {
							sink <- err
							break
						}
					}
					response.Body.Close()
				} else {
					sink <- err
				}
			} else {
				sink <- err
			}
		}(index)
	}

	// monitor transfer activity
	ticker, volumes, start, ssize, ptype := time.NewTicker(time.Second), [][2]int64{}, time.Now(), int64(-1), "start"
	for {
		select {
		case <-ticker.C:
			received, state := int64(0), RESUME{Size: size, Modified: modified, Concurrency: concurrency, Chunks: [][3]int64{}}
			for index := 0; index < concurrency; index++ {
				received += atomic.LoadInt64(&(chunks[index][2]))
				state.Chunks = append(state.Chunks, [3]int64{atomic.LoadInt64(&(chunks[index][0])),
					atomic.LoadInt64(&(chunks[index][1])), atomic.LoadInt64(&(chunks[index][2]))})
			}
			if resume != "" {
				if content, err := json.Marshal(state); err == nil {
					ioutil.WriteFile(resume, content, 0600)
				}
			}
			received = int64(math.Min(float64(received), float64(size)))
			if ssize < 0 {
				ssize = size - received
			}
			if verbose {
				volumes = append(volumes, [2]int64{received, time.Now().UnixNano() / int64(time.Millisecond)})
				if len(volumes) > 10 {
					volumes = volumes[1:]
				}
				bandwidth, elapsed, remaining := 0.0, int64(time.Now().Sub(start)/time.Second), int64(-1)
				for index := len(volumes) - 1; index >= 1; index-- {
					bandwidth += (float64((volumes[index][0] - volumes[index-1][0])) * 8) / (float64(volumes[index][1]-volumes[index-1][1]) / 1000)
				}
				if len(volumes) >= 2 {
					bandwidth /= float64(len(volumes) - 1)
				}
				if bandwidth > 0 {
					remaining = int64(math.Max(float64(elapsed), float64(ssize/int64(bandwidth/8))+1))
				}
				fmt.Fprintf(os.Stderr, "\r%d | %s/%s | %.1f%% | %s | %s/%s      ",
					concurrency, hsize(received), hsize(size), (float64(received)*100)/float64(size),
					hbandwidth(bandwidth), hduration(elapsed), hduration(remaining))
			}
			if progress {
				fmt.Printf(`{"event":"%s","concurrency":%d,"size":%d,"received":%d,"progress":%.0f,"elapsed":%d}`+"\n",
					ptype, concurrency, size, received, math.Floor((float64(received)*100)/float64(size)),
					int64(time.Now().Sub(start)/time.Second))
				ptype = "progress"
			}
			if received == size || exit {
				ticker.Stop()
				if verbose {
					fmt.Fprintf(os.Stderr, "\n")
				}
				if progress {
					fmt.Printf(`{"event":"end","concurrency":%d,"size":%d,"received":%d,"progress":100,"elapsed":%d}`+"\n",
						concurrency, size, received, int64(time.Now().Sub(start)/time.Second))
				}
				if received == size && resume != "" {
					os.Remove(resume)
				}
				os.Exit(0)
			}

		case err := <-sink:
			hfatal(fmt.Sprintf("%v", err), 4)
		}
	}
}
