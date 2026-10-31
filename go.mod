module github.com/olivere/bmecat

go 1.27.3

require (
	golang.org/x/text v0.43.0
	golang.org/x/time v0.16.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/aclements/go-moremath v0.0.0-20241023150245-c8bbc672ef66 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260813180055-c1d0aacb2297 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/perf v0.0.0-20261009192801-be2c69fb417e // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/telemetry v0.0.0-20261010193011-f71dc92da45e // indirect
	golang.org/x/tools v0.52.0 // indirect
	golang.org/x/vuln v1.9.0 // indirect
	honnef.co/go/tools v0.8.2 // indirect
	mvdan.cc/gofumpt v0.12.0 // indirect
)

tool (
	golang.org/x/perf/cmd/benchstat
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
)
