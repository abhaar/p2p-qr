.PHONY: proto
proto:
	protoc --proto_path=shared/proto \
	       --go_out=. --go_opt=module=github.com/p2p \
	       --go-grpc_out=. --go-grpc_opt=module=github.com/p2p \
	       $(shell find shared/proto -name "*.proto")

.PHONY: clean-proto
clean-proto:
	rm -rf shared/pb
