package services

import (
	// "fmt"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	jrpc "github.com/ybbus/jsonrpc"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "rpc"})

func StartServers() error {
	client := jrpc.NewClient(cmn.GetThetaRPCEndpoint())

	router, err := NewThetaRouter(client)
	if err != nil {
		logger.Fatalf("ERROR: Failed to init router: %v\n", err)
	}

	httpAddr := viper.GetString(cmn.CfgRPCHttpAddress)
	httpPort := viper.GetString(cmn.CfgRPCHttpPort)
	httpEndpoint := fmt.Sprintf("%v:%v", httpAddr, httpPort)

	logger.Infof("Started listening at: %v\n", httpEndpoint)
	if err := http.ListenAndServe(httpEndpoint, router); err != nil {
		logger.Fatalf("Theta Rosetta Adaptor server exited with error: %v\n", err)
	}

	return nil
}

func StopServers() error {
	// if httpListener != nil {
	// 	if err := httpListener.Close(); err != nil {
	// 		return err
	// 	}
	// 	httpListener = nil
	// 	logger.Infof("HTTP endpoint closed")
	// }
	// if httpHandler != nil {
	// 	httpHandler.Stop()
	// 	httpHandler = nil
	// }
	return nil
}

// NewThetaRouter returns a Mux http.Handler from a collection of
// Rosetta service controllers.
func NewThetaRouter(client jrpc.RPCClient) (http.Handler, error) {
	status, err := cmn.GetStatus(client)
	if err != nil {
		return nil, err
	}
	cmn.SetChainId(status.ChainID)
	logger.Errorf("====================== 1 chain id: %v", status.ChainID)

	asserter, err := asserter.NewServer(
		cmn.TxTypes(),
		true,
		[]*types.NetworkIdentifier{
			&types.NetworkIdentifier{
				Blockchain: "theta",
				Network:    status.ChainID,
			},
		},
		[]string{},
		false,
	)
	logger.Errorf("====================== 2 chain id: %v", status.ChainID)
	if err != nil {
		return nil, err
	}
	logger.Errorf("====================== 3 chain id: %v", status.ChainID)
	networkAPIController := server.NewNetworkAPIController(NewNetworkAPIService(client), asserter)
	accountAPIController := server.NewAccountAPIController(NewAccountAPIService(client), asserter)
	blockAPIController := server.NewBlockAPIController(NewBlockAPIService(client), asserter)
	memPoolAPIController := server.NewMempoolAPIController(NewMemPoolAPIService(client), asserter)
	// constructionAPIController := server.NewConstructionAPIController(NewConstructionAPIService(client), asserter)
	// r := server.NewRouter(networkAPIController, accountAPIController, blockAPIController, constructionAPIController)
	r := server.NewRouter(networkAPIController, accountAPIController, blockAPIController, memPoolAPIController)
	return server.CorsMiddleware(server.LoggerMiddleware(r)), nil
}
