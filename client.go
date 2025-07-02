package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
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

type clientChunk struct {
	id       int
	size     int64
	start    int64
	offset   int64
	end      int64
	request  []byte
	response []byte
	status   int
	modified int64
	etag     string
	file     *os.File
	stdout   bool
	writer   *io.PipeWriter
	data     []byte
}

var (
	clientTransport *http.Transport
	clientClient    *http.Client
	clientSize      = int64(0)
	clientReceived  = int64(0)
	clientEvent     = "start"
	clientProgress  = [32][3]int64{}
	clientResume    = ""
	clientLock      sync.Mutex
)

func clientAbort(exit int, message string) {
	if exit != 0 {
		os.Stderr.WriteString("\r                                                       \r" + message + " - aborting\n")
	}
	if Progress {
		os.Stdout.WriteString(`{"event":"error","message":"` + message + `"}` + "\n")
	}
	os.Exit(exit)
}

func clientRequest(chunk *clientChunk) (err error) {
	request, err := http.NewRequest(http.MethodGet, Flagset.Args()[0], http.NoBody)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", PROGNAME+"/"+PROGVER)
	for _, header := range Source {
		if strings.EqualFold(header[0], "host") {
			request.Host = header[1]

		} else {
			request.Header.Set(header[0], header[1])
		}
	}
	request.Header.Set("Range", "bytes="+strconv.FormatInt(chunk.offset, 10)+"-"+strconv.FormatInt(chunk.end, 10))
	if Dump {
		chunk.request, _ = httputil.DumpRequest(request, false)
	}

	response, err := clientClient.Do(request)
	if err != nil {
		return err
	}
	if Dump {
		chunk.response, _ = httputil.DumpResponse(response, false)
	}

	chunk.status = response.StatusCode
	chunk.etag = strings.TrimSpace(response.Header.Get("Etag"))
	if modified, err := time.Parse(time.RFC1123, response.Header.Get("Last-Modified")); err == nil {
		chunk.modified = modified.Unix()
	}
	if captures := rcache.Get(`^bytes (\d+)-\d+/(\d+)$`).FindStringSubmatch(response.Header.Get("Content-Range")); captures != nil && chunk.status == http.StatusPartialContent {
		chunk.offset, _ = strconv.ParseInt(captures[1], 10, 64)
		chunk.size, _ = strconv.ParseInt(captures[2], 10, 64)

	} else {
		chunk.offset, chunk.size = 0, response.ContentLength
	}
	if chunk.status/100 != 2 {
		response.Body.Close()
		return errors.New("source http status " + strconv.Itoa(chunk.status))
	}
	if chunk.size == 0 || (chunk.size < 0 && chunk.start == 0 && chunk.end == 0) {
		response.Body.Close()
		return nil
	}

	data := make([]byte, 64<<10)
	for {
		read, err := response.Body.Read(data)
		if read > 0 {
			atomic.AddInt64(&clientReceived, int64(read))
			switch {
			case chunk.file != nil:
				if chunk.start < 0 && chunk.end < 0 {
					_, err = chunk.file.Write(data[:read])

				} else {
					clientLock.Lock()
					_, err = chunk.file.WriteAt(data[:read], chunk.offset)
					clientLock.Unlock()
				}
				if err != nil {
					response.Body.Close()
					return err
				}

			case chunk.stdout:
				if _, err = os.Stdout.Write(data[:read]); err != nil {
					response.Body.Close()
					return err
				}

			case chunk.writer != nil:
				if _, err = chunk.writer.Write(data[:read]); err != nil {
					response.Body.Close()
					return err
				}

			case chunk.data != nil:
				copy(chunk.data[chunk.offset-chunk.start:], data[:read])
			}
			chunk.offset += int64(read)
			if chunk.file != nil {
				clientProgress[chunk.id][0], clientProgress[chunk.id][1], clientProgress[chunk.id][2] = chunk.start, max(chunk.start, chunk.offset-1), chunk.end
			}
		}
		if err != nil {
			response.Body.Close()
			if err != io.EOF {
				return err
			}
			if chunk.size > 0 && chunk.offset != chunk.end+1 {
				return errors.New("truncated transfer")
			}
			return nil
		}
	}
}

func Client() {
	if Flagset.NArg() < 1 {
		Flagset.Usage()
		os.Exit(1)
	}
	target := ""
	if Flagset.NArg() > 1 {
		target = Flagset.Args()[1]
	}
	clientTransport = &http.Transport{
		DialContext:           (&net.Dialer{Timeout: time.Duration(Timeout) * time.Second}).DialContext,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: Insecure},
		TLSHandshakeTimeout:   time.Duration(Timeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(Timeout) * time.Second,
		ReadBufferSize:        8 << 20,
		MaxIdleConnsPerHost:   32,
	}
	clientClient = &http.Client{Transport: clientTransport}

	chunk := clientChunk{}
	err := clientRequest(&chunk)
	if Dump {
		os.Stderr.WriteString(string(chunk.request) + string(chunk.response))
	}
	if err != nil {
		clientAbort(1, err.Error())
	}
	clientSize, clientReceived = chunk.size, 0
	if chunk.status != http.StatusPartialContent || clientSize < 0 {
		Concurrency = 1
	}
	if clientSize > 0 && clientSize/int64(Concurrency) <= 4<<20 {
		Concurrency = int(clientSize / (4 << 20))
		if clientSize%(4<<20) != 0 {
			Concurrency++
		}
	}

	var (
		file   *os.File
		reader *io.PipeReader
		writer *io.PipeWriter
	)

	waiter1, done := sync.WaitGroup{}, make(chan bool, 1)
	if target == "-" {
		Progress = false

	} else if target != "" {
		if strings.HasPrefix(target, "http") {
			method := http.MethodPut
			if Post {
				method = http.MethodPost
			}
			reader, writer = io.Pipe()
			request, err := http.NewRequest(method, target, reader)
			if err != nil {
				clientAbort(2, err.Error())
			}
			request.Header.Set("User-Agent", PROGNAME+"/"+PROGVER)
			request.Header.Set("Content-Type", "application/octet-stream")
			for _, header := range Target {
				request.Header.Set(header[0], header[1])
			}
			request.ContentLength = clientSize
			if Dump {
				dump, _ := httputil.DumpRequest(request, false)
				os.Stderr.Write(dump)
			}
			waiter1.Add(1)
			go func() {
				response, err := clientClient.Do(request)
				if err != nil {
					clientAbort(4, err.Error())
				}
				if Dump {
					dump, _ := httputil.DumpResponse(response, true)
					os.Stderr.WriteString("\r                                                            \n" + string(dump))
				}
				response.Body.Close()
				if response.StatusCode/100 != 2 {
					clientAbort(4, "target http status "+strconv.Itoa(response.StatusCode))
				}
				waiter1.Done()
			}()

		} else {
			if _, err := os.Stat(target); err != nil || Noresume {
				os.Remove(filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".resume"))
			}
			os.MkdirAll(filepath.Dir(target), 0o755)
			if clientSize < 0 {
				file, err = os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_RDWR|os.O_APPEND, 0o644)

			} else {
				file, err = os.OpenFile(target, os.O_CREATE|os.O_RDWR, 0o644)
			}
			if err != nil {
				clientAbort(2, err.Error())
			}
			if !Noresume {
				clientResume = filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".resume")
				if info, err := file.Stat(); err == nil && chunk.modified <= info.ModTime().Unix() {
					if payload, err := os.ReadFile(clientResume); err == nil {
						var progress [][3]int64

						if json.Unmarshal(payload, &progress) == nil {
							if len(progress) >= 1 && len(progress) <= 32 {
								resume := true
								for index, chunk := range progress {
									if chunk[0] < 0 || chunk[1] >= clientSize || chunk[2] >= clientSize || chunk[0] > chunk[1] || chunk[0] > chunk[2] || chunk[1] > chunk[2] ||
										(index == 0 && chunk[0] != 0) || (index == len(progress)-1 && chunk[2] != clientSize-1) || (index != 0 && chunk[0] <= progress[index-1][2]) {
										resume = false
										break
									}
								}
								if resume {
									Concurrency = len(progress)
									for index, chunk := range progress {
										clientProgress[index] = chunk
										clientReceived += chunk[1] - chunk[0] + 1
									}
								}
							}
						}
					}
				}
			}
			file.Truncate(max(0, clientSize))
		}
	}

	if Verbose || Progress {
		waiter1.Add(1)
		go func() {
			start, initial, previous, bandwidth := time.Now(), atomic.LoadInt64(&clientReceived), atomic.LoadInt64(&clientReceived), float64(0)
			for {
				received := atomic.LoadInt64(&clientReceived)
				bandwidth = float64((received - previous) * 8)
				if Verbose {
					if clientSize < 0 {
						os.Stderr.WriteString("\r" + strconv.Itoa(Concurrency) +
							" | " + utilSize(received) +
							" | " + utilBandwidth(bandwidth) +
							" | " + utilDuration(int(time.Since(start)/time.Second)) +
							"     ")

					} else {
						mbandwidth := float64((received-initial)*8) / (float64(time.Since(start)) / float64(time.Second))
						if mbandwidth == 0 {
							mbandwidth = -1
						}
						os.Stderr.WriteString("\r" + strconv.Itoa(Concurrency) +
							" | " + utilSize(received) +
							"/" + utilSize(clientSize) +
							" | " + strconv.FormatFloat(float64(received*100)/float64(clientSize), 'f', 2, 64) +
							"% | " + utilBandwidth(bandwidth) +
							" | " + utilDuration(int(time.Since(start)/time.Second)) +
							"/" + utilDuration(int(float64((clientSize-initial)*8)/mbandwidth)) +
							"     ")
					}
				}
				if received == clientSize {
					clientEvent = "end"
					bandwidth = float64((clientSize-initial)*8) / (float64(time.Since(start)) / float64(time.Second))
				}
				if Progress {
					line := `{"event":"` + clientEvent +
						`","concurrency":` + strconv.Itoa(Concurrency) +
						`,"size":` + strconv.FormatInt(clientSize, 10) +
						`,"received":` + strconv.FormatInt(clientReceived, 10) +
						`,"bandwidth":"` + utilBandwidth(bandwidth) +
						`","elapsed":` + strconv.FormatFloat(float64(time.Since(start))/float64(time.Second), 'f', 2, 64)
					if clientSize >= 0 {
						line += `,"progress":` + strconv.FormatFloat(float64(clientReceived*100)/float64(clientSize), 'f', 2, 64)
					}
					os.Stdout.WriteString(line + "}\n")
					if clientEvent == "start" {
						clientEvent = "progress"
					}
				}
				previous = received
				if clientResume != "" {
					if payload, err := json.Marshal(clientProgress[:Concurrency]); err == nil {
						os.WriteFile(clientResume, payload, 0o644)
					}
				}
				if clientSize >= 0 && received >= clientSize {
					if clientResume != "" {
						os.Remove(clientResume)
					}
					break
				}
				select {
				case <-done:
					clientSize = atomic.LoadInt64(&clientReceived)

				case <-time.After(time.Second):
				}
			}
			if Verbose {
				os.Stderr.WriteString("\r" + strconv.Itoa(Concurrency) +
					" | " + utilSize(clientSize) +
					" | " + utilBandwidth(bandwidth) +
					" | " + utilDuration(int(time.Since(start)/time.Second)) +
					"                             \n")
			}
			waiter1.Done()
		}()
	}

	if target == "" || file != nil {
		waiter2, size := sync.WaitGroup{}, clientSize/int64(Concurrency)
		for worker := 0; worker < Concurrency; worker++ {
			start, offset, end := int64(worker)*size, int64(worker)*size, min((int64(worker)*size)+size, clientSize)-1
			if worker >= Concurrency-1 {
				end = clientSize - 1
			}
			if clientSize < 0 {
				start, end = -1, -1
			}
			if clientProgress[worker][1] != 0 || clientProgress[worker][2] != 0 {
				start, offset, end = clientProgress[worker][0], clientProgress[worker][1], clientProgress[worker][2]
			}
			waiter2.Add(1)
			go func(worker int, start, offset, end int64) {
				chunk := clientChunk{id: worker, start: start, offset: offset, end: end, file: file}
				if err := clientRequest(&chunk); err != nil {
					clientAbort(3, err.Error())
				}
				waiter2.Done()
			}(worker, start, offset, end)
		}
		waiter2.Wait()
		file.Close()

	} else {
		chunks, batches, size, index := [][3]int64{}, clientSize/int64(Maxmem), int64(Maxmem/Concurrency), 0
		if clientSize%int64(Maxmem) != 0 {
			batches++
		}
		for batch := int64(0); batch < batches; batch++ {
			for worker := 0; worker < Concurrency; worker++ {
				start, offset, end := (batch*int64(Maxmem))+int64(worker)*size, (batch*int64(Maxmem))+int64(worker)*size, min((batch*int64(Maxmem))+(int64(worker)*size)+size, clientSize)-1
				if clientSize < 0 {
					start, end = -1, -1
				}
				chunks = append(chunks, [3]int64{start, offset, end})
				if end >= clientSize-1 {
					break
				}
			}
		}

		queue, received, sent := make(chan clientChunk, Concurrency), make([]*clientChunk, len(chunks)), 0
		for index = 0; index < Concurrency; index++ {
			go func(index int, start, offset, end int64) {
				chunk := clientChunk{id: index, start: start, offset: offset, end: end}
				if start < 0 && end < 0 {
					chunk.stdout, chunk.writer = target == "-", writer
				} else {
					chunk.data = bslab.Get(int(end-start+1), nil)
					chunk.data = chunk.data[:cap(chunk.data)]
				}
				if err := clientRequest(&chunk); err != nil {
					clientAbort(3, err.Error())
				}
				queue <- chunk
			}(index, chunks[index][0], chunks[index][1], chunks[index][2])
		}
		for {
			chunk := <-queue
			if chunk.stdout || chunk.writer != nil {
				break
			}
			received[chunk.id] = &chunk
			for sent < len(chunks) && received[sent] != nil {
				if target == "-" {
					if _, err := os.Stdout.Write(received[sent].data[:received[sent].end-received[sent].start+1]); err != nil {
						clientAbort(4, err.Error())
					}
				} else if writer != nil {
					if _, err := writer.Write(received[sent].data[:received[sent].end-received[sent].start+1]); err != nil {
						clientAbort(4, err.Error())
					}
				}
				bslab.Put(received[sent].data)
				sent++
			}
			if sent >= len(chunks) {
				break
			}
			if index < len(chunks) {
				go func(index int, start, offset, end int64) {
					chunk := clientChunk{id: index, start: start, offset: offset, end: end, data: bslab.Get(int(end-start+1), nil)}
					chunk.data = chunk.data[:cap(chunk.data)]
					if err := clientRequest(&chunk); err != nil {
						clientAbort(3, err.Error())
					}
					queue <- chunk
				}(index, chunks[index][0], chunks[index][1], chunks[index][2])
				index++
			}
		}
	}

	if writer != nil {
		writer.Close()
	}
	done <- true
	waiter1.Wait()
}
