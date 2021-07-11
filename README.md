## Presentation
`mfetch` is an HTTP(S)-based file transfer program written in Golang, aimed at copying data between two systems as fast as possible over multiple TCP connections.
It can act as a client or a server, and has an interrupted transfer resuming capability. In terms of performance, `mfetch` can easily saturate multi-gigabits
network paths (even with RTTs above 300ms), with a very moderate CPU usage (the main limitation being disk write-iops).


## Usage
Invoking `mfetch` without argument (or with the `--help` option) will print the following documentation:
```
$ mfetch
usage:
  mfetch [<options>] <argument(s)>

arguments:
  - client mode
        <remote-url> <local-file>
  - server mode
        <shared-folder>

options:
  -concurrency int
    	transfer concurrency level (default 10)
  -header value
    	add arbitrary HTTP header to requests (repeatable)
  -listen string
    	listening address & port in server mode (default client mode)
  -noresume
    	ignore transfer auto-resuming (default false)
  -password string
    	security password in server mode (default none)
  -progress
        emit transfer progress JSON indications (default false)
  -tls string
    	TLS certificate & key to use in server mode (or "internal", default none)
  -trustpeer
    	ignore server TLS certificate errors (default false)
  -verbose
    	verbose mode (default false)
```


## Client mode
The `remote URL` and `local target file` must be provided as arguments in client mode; if the target filename is `-`,
nothing will be saved to disk but transfer statistics will still be printed on screen if the `--verbose` option
is provided, allowing to bench the considered network path before actually transferring documents.

The following options are available in client mode:

- `--concurrency` (default 10): number of concurrent TCP connections/HTTP requests (to maximize transfer speed,
use higher values as network latency between the emitter and receiver increases).

- `--header` (default none): additionnal HTTP headers sent with all requests; can be used multiple times if needed,
for instance:
```
$ mfetch --header 'X-Header: value1' --header 'X-Another-Header: value2' https://...
```
- `--trustpeer` (default false): ignore invalid server TLS certificate (needed when using a self-signed server certificate,
like the `internal` one provided by `mfetch`, see `--tls` below).

- `--noresume` (default false): always ignore resuming state-file and restart transfer from the beginning. <ins>Note</ins>: if
the server does not support byte-range requests, `concurrency` is automatically set to 1 and transfer resuming is disabled.

- `--verbose` (default false): display transfer progress information on standard error (see format in the `Examples` section below).

- `--progress` (default false): emit transfer progress indications on standard output (in JSON format, see format in the `Examples` section below).


## Server mode
The folder to share must be provided as the only argument in server mode; only files in the specified folder will be
made accessible from an HTTP client (i.e. the potential deeper folders won't be accessible). Files starting with `.`
won't be accessible either. HTTP/2 is intentionally disabled (in HTTPS mode) to make sure connecting clients use as
many separate TCP connections as possible.

The following options are available in server mode:

- `--listen` (default none): activate `mfetch` server mode by specifying the (optional) IP address and TCP port to
listen to, for instance:
```
$ mfetch --listen 1.2.3.4:54321 ...
```

- `--tls` (default none): switch the server to HTTPS (highly recommended if it is exposed to the public Internet);
either the string `"internal"` (in which case a self-signed [internal TLS certificate](util.go#L9-L14) is used),
or a comma-separated pair of files (certificate & key PEMs), for instance:
```
$ mfetch --listen ... --tls /etc/ssl/certs/server-cert.pem,/etc/ssl/private/server-key.pem ...
```

- `--password` (default none): activate HTTP basic-authentication for all incoming requests (highly recommended if
the server is exposed to the public Internet).


## Examples
Starts an `mfetch` instance in server mode and share the files in the /tmp folder, with HTTPS and password protection
activated (you may alternatively use an existing HTTP(S) server if you already have one handy):
```
$ mfetch --listen :443 --tls internal --password password /tmp
```

Start an mfetch instance in client mode a,d download the `20GB` file from the server instance above (since the
self-signed internal TLS certificate was used, the `--trustpeer` must be added to the command-line options) :
```
$ mfetch --verbose --progress --trustpeer https://login:password@myserver.com/20GB out
6 | 860.45MB/20.0GB | 4.2% | 293.8Mb/s | 0:00:24/0:09:44
{"event":"start","concurrency":6,"size":21474836480,"received":0,"progress":0,"elapsed":0}
...
{"event":"progress","concurrency":6,"size":21474836480,"received":902247219,"progress":4,"elapsed":24}
```
When the `--verbose` option is specified on the command-line, `mfetch` will output some progress information
on the standard error in the following format:
```
<concurrency> | <transferred size>/<total size> | <transferred percentage> | <transfer speed> | <elapsed time>/<total time>
```

When the `--progress` option is specified on the command-line, `mfetch` will emit some progress indications
on the standard output in the following JSON format:
```
{"event":"<start|progress|end>","concurrency":<concurrency>,"size":<total size>,"received":<transferred size>,"progress":<transferred percentage>,"elapsed":<elapsed seconds>}
```

## Build and packaging
You need to install a recent version of the [Golang](https://golang.org/dl/) compiler (>= 1.15) and the GNU [make](https://www.gnu.org/software/make)
utility to build the `mfetch` binary. Once these requirements are fulfilled, clone the `mfetch` Github repository locally:
```
$ git clone https://github.com/pyke369/mfetch
```
and type:
```
$ make
```
This will take care of building everything. You may optionally produce a Debian binary package by typing:
```
$ make deb
```
(the [devscripts](https://packages.debian.org/fr/sid/devscripts) package needs to be installed for this last command to work)


## Projects with similar goals
- Facebook [WDT](https://github.com/facebook/wdt)


## License
MIT - Copyright (c) 2021 Pierre-Yves Kerembellec
