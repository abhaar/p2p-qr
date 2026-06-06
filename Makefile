.PHONY: proto
proto:
	protoc --proto_path=shared/proto \
	       --go_out=. --go_opt=module=github.com/p2p \
	       --go-grpc_out=. --go-grpc_opt=module=github.com/p2p \
	       $(shell find shared/proto -name "*.proto")

.PHONY: clean-proto
clean-proto:
	rm -rf shared/pb


.PHONY: sol/gen
sol/gen:
	solc \
		--abi services/blockchain/networks/evm/protocol/internal/contracts/erc20.sol \
		--overwrite \
		-o services/blockchain/networks/evm/protocol/internal/contracts

.PHONY: abi/gen
abi/gen:
	abigen \
		--abi=services/blockchain/networks/evm/protocol/internal/contracts/ERC20.abi \
		--pkg=domain \
		--type=Contract \
		--out=services/blockchain/networks/evm/protocol/internal/domain/erc20.go
