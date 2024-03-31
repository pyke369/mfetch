package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pyke369/golang-support/bslab"
	"github.com/pyke369/golang-support/rcache"
)

type CHUNK struct {
	size  int
	start int
	end   int
	data  []byte
}

var (
	transport *http.Transport
	client    *http.Client
	buckets   = [11]int64{}
	cstart    = time.Now()
	istart    = 0
	ptype     = "start"
)

func request(input *CHUNK, dump bool) (output *CHUNK, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(time.Duration(Timeout)*time.Second, func() {
		cancel()
	})
	defer timer.Stop()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, Flagset.Args()[0], nil)
	if err != nil {
		return input, err
	}
	request.Header.Set("User-Agent", fmt.Sprintf("%s/%s", PROGNAME, PROGVER))
	for _, header := range Source {
		request.Header.Set(header[0], header[1])
	}
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", input.start, input.end))
	if dump {
		headers, _ := httputil.DumpRequest(request, false)
		fmt.Fprintf(os.Stderr, "\r                                                       \r--- source request ---\n%s", headers)
	}

	response, err := client.Do(request)
	if err != nil {
		return input, err
	}
	if dump {
		headers, _ := httputil.DumpResponse(response, false)
		fmt.Fprintf(os.Stderr, "\r                                                       \r--- source response ---\n%s", headers)
	}
	captures := rcache.Get(`^bytes \d+-\d+/(\d+)$`).FindStringSubmatch(response.Header.Get("Content-Range"))
	if captures == nil || response.StatusCode != http.StatusPartialContent {
		response.Body.Close()
		return input, fmt.Errorf("source does not support partial requests")
	}
	input.size, _ = strconv.Atoi(captures[1])
	input.data = bslab.Get(input.end-input.start+1, nil)
	input.data = input.data[:cap(input.data)]
	read, offset := 0, 0
	for {
		read, err = response.Body.Read(input.data[offset:])
		timer.Reset(time.Duration(Timeout) * time.Second)
		if read > 0 {
			offset += read
		}
		if err == io.EOF {
			input.data, err = input.data[:offset], nil
			break
		}
		if err != nil {
			break
		}
	}
	response.Body.Close()
	if err == nil && (input.size <= 0 || len(input.data) != input.end-input.start+1) {
		err = fmt.Errorf("truncated source content")
	}
	if err != nil {
		bslab.Put(input.data)
	}
	return input, err
}

func hsize(size int) string {
	if size < (1 << 10) {
		return fmt.Sprintf("%dB", size)
	} else if size < (1 << 20) {
		return fmt.Sprintf("%.2fkiB", float64(size)/(1<<10))
	} else if size < (1 << 30) {
		return fmt.Sprintf("%.2fMiB", float64(size)/(1<<20))
	} else {
		return fmt.Sprintf("%.1fGiB", float64(size)/(1<<30))
	}
}
func hduration(duration int) string {
	if duration < 0 {
		return "-:--:--"
	}
	hours := duration / 3600
	duration -= (hours * 3600)
	minutes := duration / 60
	duration -= (minutes * 60)
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, duration)
}
func hbandwidth(bandwidth float64) string {
	if bandwidth < 1000 {
		return fmt.Sprintf("%.0fb/s", bandwidth)
	} else if bandwidth < (1000 * 1000) {
		return fmt.Sprintf("%.0fkb/s", bandwidth/(1000))
	} else if bandwidth < (1000 * 1000 * 1000) {
		return fmt.Sprintf("%.1fMb/s", bandwidth/(1000*1000))
	} else {
		return fmt.Sprintf("%.1fGb/s", bandwidth/(1000*1000*1000))
	}
}
func stats(start, size int, end bool) {
	for index := 10; index >= 1; index-- {
		atomic.StoreInt64(&buckets[index], atomic.LoadInt64(&buckets[index-1]))
	}
	atomic.AddInt64(&buckets[0], -atomic.LoadInt64(&buckets[1]))
	total, divider := 0, 0
	for index := 1; index <= 10; index++ {
		if buckets[index] != 0 {
			total += int(buckets[index])
			divider++
		}
	}
	if Verbose {
		bandwidth, duration, elapsed := float64(0), -1, int(time.Now().Sub(cstart)/time.Second)
		if divider != 0 {
			bandwidth = (float64(total) * 8) / (float64(divider) / 5)
			duration = int((float64(size-istart) * 8) / bandwidth)
			if duration < elapsed {
				duration = elapsed
			}
		}
		fmt.Fprintf(os.Stderr, "\r%s/%s | %.1f%% | %s | %s/%s     ", hsize(start), hsize(size),
			(float64(start)*100)/float64(size), hbandwidth(bandwidth), hduration(elapsed), hduration(duration))
		if end {
			fmt.Fprintf(os.Stderr, "\n")
		}
	}
	if Progress && ptype != "end" {
		if end {
			ptype = "end"
		}
		fmt.Printf(`{"event":"%s","size":%d,"received":%d,"progress":%.1f,"elapsed":%.3f}`+"\n",
			ptype, size, start, (float64(start)*100)/float64(size), float64(time.Now().Sub(cstart))/float64(time.Second))
		if ptype == "start" {
			ptype = "progress"
		}
	}
}

func exit(message interface{}, exit int) {
	if exit != 0 {
		fmt.Fprintf(os.Stderr, "\r                                                       \r%v - aborting\n", message)
	}
	if Progress {
		fmt.Printf(`{"event":"error","message":"%s"}`+"\n", message)
	}
	os.Exit(exit)
}

func Client() {
	var (
		file   *os.File
		reader *io.PipeReader
		writer *io.PipeWriter
	)

	if Flagset.NArg() < 1 {
		Flagset.Usage()
		os.Exit(1)
	}
	transport = &http.Transport{
		DialContext:           (&net.Dialer{Timeout: time.Duration(Timeout) * time.Second}).DialContext,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: Insecure},
		TLSHandshakeTimeout:   time.Duration(Timeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(Timeout) * time.Second,
		ReadBufferSize:        1 << 20,
		MaxIdleConnsPerHost:   32,
	}
	client = &http.Client{Transport: transport}

	source, err := request(&CHUNK{}, Dump)
	if err != nil {
		exit(err, 1)
	}
	target, start := "", 0
	if Flagset.NArg() > 1 {
		target = Flagset.Args()[1]
		if target != "-" && !strings.HasPrefix(target, "http") && !Noresume {
			if info, err := os.Stat(target); err == nil && info.Mode().IsRegular() && int(info.Size()) <= source.size {
				start = int(info.Size())
			}
		}
		if target == "-" {
			Progress = false
		}
	}
	istart = start

	queue, offset := make(chan *CHUNK, 32), start
	go func() {
		for offset < source.size {
			waiter, concurrency := &sync.WaitGroup{}, int(math.Max(1, math.Min(float64((source.size-offset)/Chunksize), float64(Concurrency))))
			for index := 0; index < concurrency; index++ {
				waiter.Add(1)
				go func(start, end int) {
					tries := 0
					for tries < Retries {
						if chunk, err := request(&CHUNK{start: start, end: end}, false); err == nil {
							queue <- chunk
							waiter.Done()
							return
						}
						tries++
					}
					queue <- &CHUNK{}
				}(offset+(Chunksize*index), int(math.Min(float64(source.size-1), float64(offset+(Chunksize*(index+1))-1))))
			}
			waiter.Wait()
			offset += Chunksize * concurrency
			offset = int(math.Min(float64(source.size), float64(offset)))
		}
		close(queue)
	}()

	go func() {
		stats(start, source.size, false)
		for range time.Tick(time.Second / 5) {
			stats(start, source.size, false)
			if start >= source.size {
				break
			}
		}
	}()

	chunks, done := map[int]*CHUNK{}, make(chan struct{})
	for start < source.size {
		chunk := <-queue
		if chunk.data == nil {
			exit("truncated source content", 2)
		}
		chunks[chunk.start] = chunk
		for {
			found := false
			for key, chunk := range chunks {
				if key == start {
					switch {
					case target == "":
					case target == "-":
						if _, err := os.Stdout.Write(chunk.data); err != nil {
							exit(err, 3)
						}
					case strings.HasPrefix(target, "http"):
						if writer == nil {
							method := http.MethodPut
							if Post {
								method = http.MethodPost
							}
							reader, writer = io.Pipe()
							request, err := http.NewRequest(method, target, reader)
							if err != nil {
								exit(err, 4)
							}
							request.Header.Set("User-Agent", fmt.Sprintf("%s/%s", PROGNAME, PROGVER))
							request.Header.Set("Content-Type", "application/octet-stream")
							for _, header := range Target {
								request.Header.Set(header[0], header[1])
							}
							request.ContentLength = int64(source.size)
							if Dump {
								headers, _ := httputil.DumpRequest(request, false)
								fmt.Fprintf(os.Stderr, "\r                                                       \r--- target request ---\n%s", headers)
							}
							go func() {
								response, err := client.Do(request)
								if err != nil {
									exit(err, 4)
								}
								if Dump {
									headers, _ := httputil.DumpResponse(response, false)
									fmt.Fprintf(os.Stderr, "\r                                                       \r--- target response ---\n%s", headers)
								}
								response.Body.Close()
								if response.StatusCode/100 != 2 {
									exit(fmt.Sprintf("invalid target status code %d", response.StatusCode), 4)
								}
								close(done)
							}()
						}
						writer.Write(chunk.data)

					default:
						if file == nil {
							os.MkdirAll(filepath.Dir(target), 0755)
							value, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
							if err != nil {
								exit(err, 5)
							}
							file = value
							file.Truncate(int64(start))
						}
						if _, err := file.Write(chunk.data); err != nil {
							exit(err, 5)
						}
					}
					atomic.AddInt64(&buckets[0], int64(len(chunk.data)))
					bslab.Put(chunk.data)
					start = chunk.end + 1
					delete(chunks, key)
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
	}
	if file != nil {
		file.Close()
	}
	if writer != nil {
		<-done
		writer.Close()
	}
	stats(start, source.size, true)
}
