module github.com/aamoghS/sideprojects/hotwire

go 1.22.0

toolchain go1.24.4

require (
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
	github.com/aamoghS/sideprojects/simhttp v0.0.0
)

replace (
	github.com/aamoghS/sideprojects/minstd => ../minstd
	github.com/aamoghS/sideprojects/simhttp => ../simhttp
)

require (
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	github.com/aamoghS/sideprojects/minstd v0.0.0 // indirect
)
