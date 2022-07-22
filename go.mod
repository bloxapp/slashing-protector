module github.com/bloxapp/slashing-protector

go 1.18

require (
	github.com/alecthomas/kong v0.2.22
	github.com/attestantio/go-eth2-client v0.8.2
	github.com/carlmjohnson/requests v0.21.12
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-chi/render v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/prysmaticlabs/prysm v1.4.4
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.21.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.1 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/aristanetworks/goarista v0.0.0-20200805130819-fd197cf57d96 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ethereum/go-ethereum v1.10.17-0.20220323200026-535f25d65fa0 // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/ferranbt/fastssz v0.0.0-20210905181407-59cf6761a7d5 // indirect
	github.com/goccy/go-yaml v1.8.9 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.0.1 // indirect
	github.com/k0kubun/go-ansi v0.0.0-20180517002512-3bf9e2903213 // indirect
	github.com/klauspost/cpuid/v2 v2.0.12 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/prysmaticlabs/go-bitfield v0.0.0-20210809151128-385d8c5e3fb7 // indirect
	github.com/prysmaticlabs/prombbolt v0.0.0-20210126082820-9b7adba6db7c // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/schollz/progressbar/v3 v3.3.4 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/thomaso-mirodin/intmath v0.0.0-20160323211736-5dc6d854e46e // indirect
	github.com/uber/jaeger-client-go v2.25.0+incompatible // indirect
	github.com/urfave/cli/v2 v2.3.0 // indirect
	go.etcd.io/bbolt v1.3.5 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/crypto v0.0.0-20220321153916-2c7772ba3064 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.34.0 // indirect
	google.golang.org/genproto v0.0.0-20210426193834-eac7f76ac494 // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

// "go.mode throwing error when prysm point dirctly to v2.X so replace to v2.0.1 here"
replace github.com/prysmaticlabs/prysm => github.com/prysmaticlabs/prysm v1.4.2-0.20220616131429-4de92bafc4bb

replace github.com/ferranbt/fastssz => github.com/prysmaticlabs/fastssz v0.0.0-20220110145812-fafb696cae88
