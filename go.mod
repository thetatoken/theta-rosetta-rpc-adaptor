module github.com/thetatoken/theta-rosetta-rpc-adaptor

require (
	github.com/coinbase/rosetta-sdk-go v0.6.10
	github.com/dgraph-io/badger v1.6.1 // indirect
	github.com/ethereum/go-ethereum v1.9.25 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/golang/snappy v0.0.3-0.20201103224600-674baa8c7fc3 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954 // indirect
	github.com/thetatoken/theta v0.0.0
	github.com/thetatoken/theta/common v0.0.0
	github.com/ybbus/jsonrpc v2.1.2+incompatible
	golang.org/x/net v0.0.0-20210220033124-5f55cee0dc0d // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
)

replace github.com/thetatoken/theta v0.0.0 => ../theta

replace github.com/thetatoken/theta/common v0.0.0 => ../theta/common

replace github.com/thetatoken/theta/rpc/lib/rpc-codec/jsonrpc2 v0.0.0 => ../theta/rpc/lib/rpc-codec/jsonrpc2/

replace github.com/ethereum/go-ethereum => github.com/ethereum/go-ethereum v1.9.9

go 1.13
