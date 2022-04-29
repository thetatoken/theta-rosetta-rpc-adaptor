package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"
	jrpc "github.com/ybbus/jsonrpc"
)

type GetPendingTransactionsArgs struct {
}

type GetPendingTransactionsResult struct {
	TxHashes []string `json:"tx_hashes"`
}

type memPoolAPIService struct {
	client jrpc.RPCClient
}

// NewMemPoolAPIService creates a new instance of an MemPoolAPIService.
func NewMemPoolAPIService(client jrpc.RPCClient) server.MempoolAPIServicer {
	return &memPoolAPIService{
		client: client,
	}
}

// MemPool implements the /mempool endpoint.
func (s *memPoolAPIService) Mempool(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.MempoolResponse, *types.Error) {
	if !strings.EqualFold(cmn.CfgRosettaModeOnline, viper.GetString(cmn.CfgRosettaMode)) {
		return nil, cmn.ErrUnavailableOffline
	}

	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	rpcRes, rpcErr := s.client.Call("theta.GetPendingTransactions", GetPendingTransactionsArgs{})

	parse := func(jsonBytes []byte) (interface{}, error) {
		pendingTxs := GetPendingTransactionsResult{}
		json.Unmarshal(jsonBytes, &pendingTxs)

		resp := types.MempoolResponse{}
		resp.TransactionIdentifiers = make([]*types.TransactionIdentifier, 0)
		for _, txHash := range pendingTxs.TxHashes {
			txId := types.TransactionIdentifier{
				Hash: txHash,
			}
			resp.TransactionIdentifiers = append(resp.TransactionIdentifiers, &txId)
		}

		return resp, nil
	}

	res, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		return nil, cmn.ErrUnableToGetMemPool
	}

	ret, _ := res.(types.MempoolResponse)
	return &ret, nil
}

// MempoolTransaction implements the /mempool/transaction endpoint.
func (s *memPoolAPIService) MempoolTransaction(
	ctx context.Context,
	request *types.MempoolTransactionRequest,
) (*types.MempoolTransactionResponse, *types.Error) {
	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	rpcRes, rpcErr := s.client.Call("theta.GetTransaction", GetTransactionArgs{
		Hash: request.TransactionIdentifier.Hash,
	})

	parse := func(jsonBytes []byte) (interface{}, error) {
		txResult := GetTransactionResult{}
		json.Unmarshal(jsonBytes, &txResult)

		resp := types.MempoolTransactionResponse{}

		var objMap map[string]json.RawMessage
		json.Unmarshal(jsonBytes, &objMap)
		if objMap["transaction"] != nil {
			var rawTx json.RawMessage
			json.Unmarshal(objMap["transaction"], &rawTx)
			status := string(txResult.Status)
			if "not_found" != status {
				tx := cmn.ParseTx(cmn.TxType(txResult.Type), rawTx, txResult.TxHash, &status)
				resp.Transaction = &tx
			}
		}
		if resp.Transaction == nil {
			return nil, fmt.Errorf("%v", cmn.ErrUnableToGetBlkTx)
		}
		return resp, nil
	}

	res, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		return nil, cmn.ErrUnableToGetBlkTx
	}

	ret, _ := res.(types.MempoolTransactionResponse)
	return &ret, nil
}
