package common

import (
	"github.com/spf13/viper"
)

const (
	// CfgConfigPath defines custom config path
	CfgConfigPath = "config.path"

	// CfgThetaRPCEndpoint configures the Theta RPC endpoint
	CfgThetaRPCEndpoint = "theta.rpcEndpoint"

	// CfgRPCHttpAddress sets the binding address of RPC http service.
	CfgRPCHttpAddress = "rpc.httpAddress"
	// CfgRPCHttpPort sets the port of RPC http service.
	CfgRPCHttpPort = "rpc.httpPort"
	// CfgRPCWSAddress sets the binding address of RPC websocket service.
	CfgRPCWSAddress = "rpc.wsAddress"
	// CfgRPCWSPort sets the port of RPC websocket service.
	CfgRPCWSPort = "rpc.wsPort"
	// CfgRPCMaxConnections limits concurrent connections accepted by RPC server.
	CfgRPCMaxConnections = "rpc.maxConnections"
	// CfgRPCTimeoutSecs set a timeout for RPC.
	CfgRPCTimeoutSecs = "rpc.timeoutSecs"

	// CfgLogLevels sets the log level.
	CfgLogLevels = "log.levels"
	// CfgLogPrintSelfID determines whether to print node's ID in log (Useful in simulation when
	// there are more than one node running).
	CfgLogPrintSelfID = "log.printSelfID"

	CfgRosettaVersion = "rosetta.version"

	// CfgRosettaMode determines if the implementation is permitted to make outbound connections.
	CfgRosettaMode        = "rosetta.mode"
	CfgRosettaModeOnline  = "online"
	CfgRosettaModeOffline = "offline"
)

func init() {
	viper.SetDefault(CfgThetaRPCEndpoint, "http://127.0.0.1:16888/rpc")

	viper.SetDefault(CfgRPCHttpAddress, "0.0.0.0")
	viper.SetDefault(CfgRPCHttpPort, "8080")
	viper.SetDefault(CfgRPCWSAddress, "0.0.0.0")
	viper.SetDefault(CfgRPCWSPort, "8081")
	viper.SetDefault(CfgRPCMaxConnections, 2048)
	viper.SetDefault(CfgRPCTimeoutSecs, 600)

	viper.SetDefault(CfgLogLevels, "*:debug")
	viper.SetDefault(CfgLogPrintSelfID, false)

	viper.SetDefault(CfgRosettaVersion, "1.1.1")
	viper.SetDefault(CfgRosettaMode, "online")
}
