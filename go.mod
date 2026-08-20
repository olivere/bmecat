module github.com/olivere/bmecat

go 1.25.0

require (
	golang.org/x/text v0.41.0
	golang.org/x/time v0.15.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/aclements/go-moremath v0.0.0-20241023150245-c8bbc672ef66 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/perf v0.0.0-20260709024250-82a0b07e230d // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/telemetry v0.0.0-20260708182218-49f421fb7959 // indirect
	golang.org/x/tools v0.48.0 // indirect
	golang.org/x/vuln v1.5.0 // indirect
	honnef.co/go/tools v0.7.0 // indirect
	mvdan.cc/gofumpt v0.10.0 // indirect
)

tool (
	golang.org/x/perf/cmd/benchstat
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
)
