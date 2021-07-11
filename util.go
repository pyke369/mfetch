package main

import (
	"fmt"
	"os"
	"strings"
)

// internal self-signed X509 TLS certificate
// Serial:     58:c4:a8:c1:27:45:86:c0:e1:52:16:ab:d3:e4:3d:72:fb:b5:86:09
// Issuer:     CN = mfetch
// Subject:    CN = mfetch
// Not Before: Jan 21 15:04:26 2021 GMT
// Not After : Jan 19 15:04:26 2031 GMT
var tlscert, tlskey = `-----BEGIN CERTIFICATE-----
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
`

// repeated command-line option handler
type multiflag [][2]string

func (m *multiflag) String() string {
	return ""
}
func (m *multiflag) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) >= 2 {
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(strings.Join(parts[1:], ":"))
		if len(key) > 0 && len(value) > 0 {
			*m = append(*m, [2]string{key, value})
		}
	}
	return nil
}

// duration human-display
func hduration(duration int64) string {
	if duration < 0 {
		return "-:--:--"
	}
	hours := duration / 3600
	duration -= (hours * 3600)
	minutes := duration / 60
	duration -= (minutes * 60)
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, duration)
}

// size human-display
func hsize(size int64) string {
	if size < (1 << 10) {
		return fmt.Sprintf("%dB", size)
	} else if size < (1 << 20) {
		return fmt.Sprintf("%.2fkB", float64(size)/(1<<10))
	} else if size < (1 << 30) {
		return fmt.Sprintf("%.2fMB", float64(size)/(1<<20))
	} else {
		return fmt.Sprintf("%.1fGB", float64(size)/(1<<30))
	}
}

// bandwidth human-display
func hbandwidth(bandwidth float64) string {
	if bandwidth < 1000 {
		return fmt.Sprintf("%.0fb/s", bandwidth)
	} else if bandwidth < 1000000 {
		return fmt.Sprintf("%.0fkb/s", bandwidth/1000)
	} else if bandwidth < 1000000000 {
		return fmt.Sprintf("%.1fMb/s", bandwidth/1000000)
	} else {
		return fmt.Sprintf("%.1fGb/s", bandwidth/1000000000)
	}
}

// error human-display & abort
func hfatal(message string, exit int) {
	if exit != 0 {
		fmt.Fprintf(os.Stderr, "\r                                                  \r%s - aborting\n", message)
	}
	if progress {
		fmt.Printf(`{"event":"error","message":"%s"}`+"\n", message)
	}
	os.Exit(exit)
}
