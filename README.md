## Presentation
`mfetch` is an HTTP(S)-based file transfer program written in Golang, aimed at copying data between systems as fast
as possible over multiple TCP connections.  It can act as a client or a server, and has an interrupted transfer resuming
capability. In terms of performance, `mfetch` can easily saturate multi-gigabits network paths (even with back-to-back
latencies above 300ms), with a very moderate CPU usage (the main limitation being disk write iops).


## Usage
Invoking `mfetch` without argument (or with `-help`) will print the following documentation:
```
$ mfetch
usage:
  mfetch [<option...>] <argument...>

arguments:
  - client mode
        <source-url> [-|<local-file>|<target-url>]
  - server mode
        [<local-folder>]

options:
  -certificate string
        use provided TLS certificate & key in server mode (or "internal", no default)
  -concurrency int
        set transfer concurrency level (default 8)
  -dump
        dump HTTP requests and responses (default false)
  -insecure
        ignore remote TLS certificate errors (default false)
  -listen string
        set listening address & port in server mode (default client mode)
  -noresume
        disable transfer auto-resuming (default false)
  -password string
        set security password in server mode (no default)
  -post
        use HTTP POST method for remote target (default PUT)
  -progress
        emit transfer progress JSON indications (default false)
  -retries int
        set source request retries (default 4)
  -source value
        add HTTP header to source request (repeatable, no default)
  -target value
        add HTTP header to target request (repeatable, no default)
  -timeout int
        set requests timeout (default 10)
  -verbose
        set verbose mode (default false)
```


## Client mode
A valid `source-url` argument must be provided in client mode; if no target argument is provided, nothing will be saved to disk (or written to remote target), but transfer statistics will still be printed on screen if the `-verbose` option is provided, allowing to bench the considered network path before actually transferring documents.

The following options are available in client mode:

- `-concurrency` (default `6`): number of concurrent TCP connections/HTTP requests (may be increased to maximize transfer aggregated speed, as network latency between the client and server also increases).

- `-dump` (default `false`): dump requests and responses on standard error (mainly for debugging purpose).

- `-insecure` (default `false`): ignore invalid server TLS certificate (needed when using a self-signed server certificate, like the `internal` one provided by `mfetch`, see `-certificate` below).

- `-noresume` (default `false`): always restart transfer from the beginning. <ins>Note</ins>: if the server does not support byte-range requests, `concurrency` is automatically set to 1 and transfer resuming is disabled.

- `-post` (default `PUT`): use POST method (instead of PUT) in the remote `target-url` request.

- `-progress` (default `false`): emit transfer progress indications on standard output (in JSON format, see format in the `Examples` section below).

- `-retries` (default `4`): retry failed transfers for the specified number of times.

- `-source` (`no default`): additionnal HTTP headers sent with all source requests; can be used multiple times if needed, for instance:
```
$ mfetch -source 'X-Header: value1' -source 'X-Another-Header: value2' https://...
```
- `-target` (`no default`): additionnal HTTP headers sent with the target request; can be used multiple times if needed, for instance:
```
$ mfetch -target 'X-Header: value1' -target 'X-Another-Header: value2' https://...
```
- `-timeout` (default `10s`): fail a connection if data transfer stay stalled for the specified number of seconds (also see `retries` above).

- `-verbose` (default `false`): display transfer progress information on standard error (see format in the `Examples` section below).


## Server mode
A `local-folder` argument may be provided in server mode, in which case only files from the specified folder will be made accessible from an HTTP client (deeper folders won't be accessible). Files starting with `.` won't be accessible either. If `mfetch` is started with no argument, it will server virtual files with sizes based on their names, for benchmarking purpose (see syntax in the `Examples` section below). HTTP/2 is intentionally disabled (in HTTPS mode) to make sure connecting clients use as many separate TCP connections as possible.

The following options are available in server mode:

- `-listen` (`no default`): activate `mfetch` server mode by specifying the (optional) IP address and TCP port to listen to, for instance:
```
$ mfetch -listen 1.2.3.4:54321 ...
```
- `-certificate` (`no default`): switch the server to HTTPS (highly recommended if exposed to the public Internet); either the string `"internal"` (in which case a self-signed [internal TLS certificate](server.go#L24-L29) is used), or a comma-separated pair of files (certificate & key PEMs), for instance:
```
$ mfetch -listen ... -certificate /etc/ssl/certs/server-cert.pem,/etc/ssl/private/server-key.pem ...
```
- `-password` (`no default`): activate HTTP basic-authentication for all incoming requests (highly recommended if the server is exposed to the public Internet).

- `-dump` (default `false`): dump requests and responses statistics on standard error.

- `-verbose` (default `false`): display in-flight requests count and total egress bandwidth on standard error.

## Examples
Starts an `mfetch` instance in "virtual files" server mode; clients requests matching the `/\d+[KMG]?i?B?` regex pattern (for instance `/10M`, `/3GiB` or `/654321`) will be honoured by serving an all-zeroed content of the corresponding size:
```
$ mfetch -listen :8000
```

Starts an `mfetch` instance in server mode and share the files in the /tmp folder, with HTTPS and password protection activated (you may alternatively use an existing HTTP(S) server if you already have one handy):
```
$ mfetch -listen :443 -certificate internal -password password /tmp
```

Start an mfetch instance in client mode and download the `20GiB` file from the server instance above (since the self-signed internal TLS certificate was used, the `-insecure` must be added to the command-line options for this to work) :
```
$ mfetch -verbose -progress -insecure https://login:password@localhost/20GiB out
860.45MB/20.0GB | 4.2% | 293.8Mb/s | 0:00:24/0:09:44
{"event":"start","size":21474836480,"received":0,"progress":0,"elapsed":0.000}
...
{"event":"progress","size":21474836480,"received":902247219,"progress":4,"elapsed":24.287}
```
When the `-verbose` option is specified on the command-line, `mfetch` will output some progress information on the standard error in the following format:
```
<received size>/<total size> | <received percentage> | <transfer speed> | <elapsed time>/<estimated total time>
```

When the `-progress` option is specified on the command-line, `mfetch` will emit some progress indications on the standard output in the following JSON format:
```
{"event":"start|progress|end","size":<total size>,"received":<transferred size>,"progress":<transferred percentage>,"elapsed":<elapsed seconds>}
```

## Build
You need to install a recent version of the [Golang](https://golang.org/dl/) compiler (>= 1.22) and the GNU [make](https://www.gnu.org/software/make)
utility to build the `mfetch` binary. Once these requirements are fulfilled, clone the `mfetch` Github repository locally:
```
$ git clone https://github.com/pyke369/mfetch
```
and type:
```
$ make
```
This will take care of building everything. You may alternatively use the Golang toolchain and install `mfetch` locally with the following command:
```
go install github.com/pyke369/mfetch@lastest
```

## Projects with similar goals
- Facebook [WDT](https://github.com/facebook/wdt)


## License
MIT - Copyright (c) 2019-2024 Pierre-Yves Kerembellec
