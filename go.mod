module github.com/thetatoken/theta-rosetta-rpc-adaptor

require (
	github.com/coinbase/rosetta-sdk-go v0.6.10
	github.com/dgraph-io/badger v1.6.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/thetatoken/theta v0.0.0
	github.com/thetatoken/theta/common v0.0.0
	github.com/ybbus/jsonrpc v2.1.2+incompatible
)

replace github.com/thetatoken/theta v0.0.0 => ../theta

replace github.com/thetatoken/theta/common v0.0.0 => ../theta/common

replace github.com/thetatoken/theta/rpc/lib/rpc-codec/jsonrpc2 v0.0.0 => ../theta/rpc/lib/rpc-codec/jsonrpc2/

replace github.com/ethereum/go-ethereum => github.com/ethereum/go-ethereum v1.10.9

go 1.13
