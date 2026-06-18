module github.com/olivere/bmecat

go 1.25.0

require (
	golang.org/x/text v0.38.0
	golang.org/x/time v0.15.0
)

require (
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/telemetry v0.0.0-20260617140237-9b6dc03d9327 // indirect
	golang.org/x/tools v0.46.0 // indirect
	golang.org/x/vuln v1.4.0 // indirect
	mvdan.cc/gofumpt v0.10.0 // indirect
)

tool (
	golang.org/x/vuln/cmd/govulncheck
	mvdan.cc/gofumpt
)
