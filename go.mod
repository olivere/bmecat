module github.com/olivere/bmecat

go 1.25.0

require (
	golang.org/x/text v0.38.0
	golang.org/x/time v0.15.0
)

require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/aclements/go-moremath v0.0.0-20210112150236-f10218a38794 // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/perf v0.0.0-20260615155930-9e4b9ddef5b6 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/telemetry v0.0.0-20260617140237-9b6dc03d9327 // indirect
	golang.org/x/tools v0.46.0 // indirect
	golang.org/x/vuln v1.4.0 // indirect
	honnef.co/go/tools v0.7.0 // indirect
	mvdan.cc/gofumpt v0.10.0 // indirect
)

tool (
	golang.org/x/perf/cmd/benchstat
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
)
