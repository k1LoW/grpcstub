module github.com/k1LoW/grpcstub

go 1.24.0

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1
	connectrpc.com/connect v1.19.1
	github.com/IGLOU-EU/go-wildcard/v2 v2.1.0
	github.com/bmatcuk/doublestar/v4 v4.9.1
	github.com/bufbuild/protocompile v0.14.1
	github.com/google/go-cmp v0.7.0
	github.com/jaswdr/faker v1.19.1
	github.com/jhump/protoreflect/v2 v2.0.0-beta.2
	github.com/k1LoW/bufresolv v0.7.10
	github.com/k1LoW/protoresolv v0.1.8
	github.com/tenntenn/golden v0.5.5
	golang.org/x/net v0.48.0
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/josharian/mapfs v0.0.0-20210615234106-095c008854e6 // indirect
	github.com/josharian/txtarfs v0.0.0-20240408113805-5dc76b8fe6bf // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Licensing error. ref: https://github.com/k1LoW/grpcstub/issues/182
retract [v0.8.0, v0.25.12]
