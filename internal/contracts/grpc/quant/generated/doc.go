// Package quantv1 will hold the generated protobuf/gRPC Go code for
// ../quant_engine.proto once the Rust Quant Engine exists and `make proto`
// has been run in an environment with protoc, protoc-gen-go and
// protoc-gen-go-grpc installed. It is intentionally empty in this delivery:
// this repository's toolchain does not have protoc available, and the
// application does not need generated code today — the
// application/ports/output.QuantEngine Go interface is satisfied by an
// in-process fake (internal/infrastructure/external/quant/grpc) instead.
//
// To generate:
//
//	protoc \
//	  --go_out=. --go_opt=module=trading-core \
//	  --go-grpc_out=. --go-grpc_opt=module=trading-core \
//	  internal/contracts/grpc/quant/quant_engine.proto
package quantv1
