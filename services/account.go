package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	jrpc "github.com/ybbus/jsonrpc"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"
	ttypes "github.com/thetatoken/theta/ledger/types"
)

// var logger *log.Entry = log.WithFields(log.Fields{"prefix": "account"})

type GetAccountArgs struct {
	Address string `json:"address"`
	// Height  JSONUint64 `json:"height"`
	Preview bool `json:"preview"` // preview the account balance from the ScreenedView
}

type GetAccountResult struct {
	*ttypes.Account
	Address string `json:"address"`
}

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
	// terr := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier)
	// if terr != nil {
	// 	return nil, terr
	// }

	// var height cmn.JSONUint64
	// if request.BlockIdentifier == nil {
	// 	height = 0
	// } else {
	// 	height = cmn.JSONUint64(*request.BlockIdentifier.Index)
	// }

	status, err := GetStatus(s.client)

	rpcRes, rpcErr := s.client.Call("theta.GetAccount", GetAccountArgs{
		Address: request.AccountIdentifier.Address,
		// Height:  height,
	})

	parse := func(jsonBytes []byte) (interface{}, error) {
		account := GetAccountResult{}.Account
		json.Unmarshal(jsonBytes, &account)

		resp := types.AccountBalanceResponse{}
		if request.BlockIdentifier != nil {
			resp.BlockIdentifier = &types.BlockIdentifier{Index: *request.BlockIdentifier.Index, Hash: *request.BlockIdentifier.Hash}
		} else {
			resp.BlockIdentifier = &types.BlockIdentifier{Index: int64(status.LatestFinalizedBlockHeight), Hash: status.LatestFinalizedBlockHash.String()}
		}
		resp.Metadata = map[string]interface{}{"sequence_number": account.Sequence}

		var needTheta, needTFuel bool
		if request.Currencies != nil {
			for _, currency := range request.Currencies {
				if strings.EqualFold(currency.Symbol, cmn.Theta) {
					needTheta = true
				} else if strings.EqualFold(currency.Symbol, cmn.TFuel) {
					needTFuel = true
				}
			}
		} else {
			needTheta = true
			needTFuel = true
		}

		if needTheta {
			var thetaBalance types.Amount
			thetaBalance.Value = account.Balance.ThetaWei.String()
			thetaBalance.Currency = &types.Currency{Symbol: cmn.Theta, Decimals: cmn.CoinDecimals}
			resp.Balances = append(resp.Balances, &thetaBalance)
		}

		if needTFuel {
			var tfuelBalance types.Amount
			tfuelBalance.Value = account.Balance.TFuelWei.String()
			tfuelBalance.Currency = &types.Currency{Symbol: cmn.TFuel, Decimals: cmn.CoinDecimals}
			resp.Balances = append(resp.Balances, &tfuelBalance)
		}

		return resp, nil
	}

	res, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		return nil, cmn.ErrUnableToGetAccount
	}

	ret, _ := res.(types.AccountBalanceResponse)
	return &ret, nil
}

// AccountCoins implements the /account/coins endpoint.
func (s *accountAPIService) AccountCoins(
	ctx context.Context,
	request *types.AccountCoinsRequest,
) (*types.AccountCoinsResponse, *types.Error) {
	status, err := GetStatus(s.client)

	rpcRes, rpcErr := s.client.Call("theta.GetAccount", GetAccountArgs{
		Address: request.AccountIdentifier.Address,
		// Height:  status.LatestFinalizedBlockHeight,
	})

	parse := func(jsonBytes []byte) (interface{}, error) {
		account := GetAccountResult{}.Account
		json.Unmarshal(jsonBytes, &account)

		resp := types.AccountCoinsResponse{}
		resp.BlockIdentifier = &types.BlockIdentifier{Index: int64(status.LatestFinalizedBlockHeight), Hash: status.LatestFinalizedBlockHash.String()}
		resp.Metadata = map[string]interface{}{"sequence_number": account.Sequence}

		var needTheta, needTFuel bool
		if request.Currencies != nil {
			for _, currency := range request.Currencies {
				if strings.EqualFold(currency.Symbol, cmn.Theta) {
					needTheta = true
				} else if strings.EqualFold(currency.Symbol, cmn.TFuel) {
					needTFuel = true
				}
			}
		} else {
			needTheta = true
			needTFuel = true
		}

		if needTheta {
			var thetaCoin types.Coin
			thetaCoin.CoinIdentifier = &types.CoinIdentifier{Identifier: "theta"} //TODO ?
			var thetaBalance types.Amount
			thetaBalance.Value = account.Balance.ThetaWei.String()
			thetaBalance.Currency = &types.Currency{Symbol: cmn.Theta, Decimals: cmn.CoinDecimals}
			thetaCoin.Amount = &thetaBalance
			resp.Coins = append(resp.Coins, &thetaCoin)
		}

		if needTFuel {
			var tfuelCoin types.Coin
			tfuelCoin.CoinIdentifier = &types.CoinIdentifier{Identifier: "tfuel"} //TODO ?
			var tfuelBalance types.Amount
			tfuelBalance.Value = account.Balance.TFuelWei.String()
			tfuelBalance.Currency = &types.Currency{Symbol: cmn.TFuel, Decimals: cmn.CoinDecimals}
			tfuelCoin.Amount = &tfuelBalance
			resp.Coins = append(resp.Coins, &tfuelCoin)
		}

		return resp, nil
	}

	res, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		return nil, cmn.ErrUnableToGetAccount
	}

	ret, _ := res.(types.AccountCoinsResponse)
	return &ret, nil
}
