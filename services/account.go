package services

import (
	"context"
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	jrpc "github.com/ybbus/jsonrpc"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"
)

// var logger *log.Entry = log.WithFields(log.Fields{"prefix": "account"})

type accountAPIService struct {
	client jrpc.RPCClient
}

// NewAccountAPIService creates a new instance of an AccountAPIService.
func NewAccountAPIService(client jrpc.RPCClient) server.AccountAPIServicer {
	return &accountAPIService{
		client: client,
	}
}

// AccountBalance implements the /account/balance endpoint.
func (s *accountAPIService) AccountBalance(
	ctx context.Context,
	request *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	// terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier)
	// if terr != nil {
	// 	return nil, terr
	// }

	// var height cmn.JSONUint64
	// if request.BlockIdentifier == nil {
	// 	height = 0
	// } else {
	// 	height = cmn.JSONUint64(*request.BlockIdentifier.Index)
	// }

	rpcRes, rpcErr := s.client.Call("theta.GetAccount", cmn.GetAccountArgs{
		Address: request.AccountIdentifier.Address,
		// Height:  height,
	})

	parse := func(jsonBytes []byte) (interface{}, error) {
		account := cmn.GetAccountResult{}.Account
		json.Unmarshal(jsonBytes, &account)

		resp := types.AccountBalanceResponse{}
		if request.BlockIdentifier != nil {
			resp.BlockIdentifier = &types.BlockIdentifier{Index: *request.BlockIdentifier.Index, Hash: *request.BlockIdentifier.Hash}
			// resp.BlockIdentifier.Index = *request.BlockIdentifier.Index
			// resp.BlockIdentifier.Hash = *request.BlockIdentifier.Hash
		}
		resp.Metadata = map[string]interface{}{"sequence_number": account.Sequence}

		var thetaBalance types.Amount
		thetaBalance.Value = account.Balance.ThetaWei.String()
		thetaBalance.Currency = &types.Currency{Symbol: cmn.Theta, Decimals: cmn.CoinDecimals}
		resp.Balances = append(resp.Balances, &thetaBalance)

		var tfuelBalance types.Amount
		tfuelBalance.Value = account.Balance.TFuelWei.String()
		tfuelBalance.Currency = &types.Currency{Symbol: cmn.TFuel, Decimals: cmn.CoinDecimals}
		resp.Balances = append(resp.Balances, &tfuelBalance)

		return resp, nil
	}

	res, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		return nil, nil // ErrUnableToGetAccount
	}

	ret, _ := res.(types.AccountBalanceResponse)
	return &ret, nil
}

// AccountCoins implements the /account/coins endpoint.
func (s *accountAPIService) AccountCoins(
	ctx context.Context,
	request *types.AccountCoinsRequest,
) (*types.AccountCoinsResponse, *types.Error) {
	return nil, nil
}
