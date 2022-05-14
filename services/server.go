package services

import (
	// "fmt"
	"errors"
	"fmt"
	"net/http"
	"os"

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

	asserter, err := asserter.NewServer(
		cmn.TxOpTypes(),
		true,
		[]*types.NetworkIdentifier{
			{
				Blockchain: cmn.ChainName,
				Network:    status.ChainID,
			},
		},
		[]string{},
		false,
	)
	if err != nil {
		return nil, err
	}

	needQueryReturnStakes := false
	if _, err := os.Stat("/data/return_stakes"); errors.Is(err, os.ErrNotExist) {
		needQueryReturnStakes = true
	}

	db, err := cmn.NewLDBDatabase("/data/return_stakes", 64, 0)
	stakeService := cmn.NewStakeService(client, db)

	// populate kvstore for vcp/gcp/eenp stakes having withdrawn:true
	if needQueryReturnStakes {
		stakeService.GenStakesForSnapshot()
	}

	// //temp
	// iter := db.NewIterator()
	// for iter.Next() {
	// 	key := iter.Key()
	// 	returnStakeTxs := cmn.ReturnStakeTxs{}
	// 	kvstore := cmn.NewKVStore(db)
	// 	kvstore.Get(key, &returnStakeTxs)
	// }
	// iter.Release()
	// err = iter.Error()

	networkAPIController := server.NewNetworkAPIController(NewNetworkAPIService(client), asserter)
	accountAPIController := server.NewAccountAPIController(NewAccountAPIService(client), asserter)
	blockAPIController := server.NewBlockAPIController(NewBlockAPIService(client, db, stakeService), asserter)
	memPoolAPIController := server.NewMempoolAPIController(NewMemPoolAPIService(client), asserter)
	constructionAPIController := server.NewConstructionAPIController(NewConstructionAPIService(client), asserter)
	r := server.NewRouter(networkAPIController, accountAPIController, blockAPIController, memPoolAPIController, constructionAPIController)
	return server.CorsMiddleware(server.LoggerMiddleware(r)), nil
}
