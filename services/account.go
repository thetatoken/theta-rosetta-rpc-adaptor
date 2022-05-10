package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/spf13/viper"
	jrpc "github.com/ybbus/jsonrpc"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"
	"github.com/thetatoken/theta/common"
	ttypes "github.com/thetatoken/theta/ledger/types"
)

// var logger *log.Entry = log.WithFields(log.Fields{"prefix": "account"})

type GetAccountArgs struct {
	Address string            `json:"address"`
	Height  common.JSONUint64 `json:"height"`
	Preview bool              `json:"preview"` // preview the account balance from the ScreenedView
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
	if !strings.EqualFold(cmn.CfgRosettaModeOnline, viper.GetString(cmn.CfgRosettaMode)) {
		return nil, cmn.ErrUnavailableOffline
	}

	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	var blockHeight common.JSONUint64
	var blockHash string
	if request.BlockIdentifier == nil {
		status, err := cmn.GetStatus(s.client)
		if err != nil {
			return nil, cmn.ErrUnableToGetAccount
		}
		blockHeight = status.LatestFinalizedBlockHeight
		blockHash = status.LatestFinalizedBlockHash.String()
	} else {
		if request.BlockIdentifier.Index != nil && request.BlockIdentifier.Hash != nil {
			blockHeight = common.JSONUint64(*request.BlockIdentifier.Index)
			blockHash = *request.BlockIdentifier.Hash
		} else if request.BlockIdentifier.Index != nil {
			blockHeight = common.JSONUint64(*request.BlockIdentifier.Index)
			blk, err := cmn.GetBlockIdentifierByHeight(s.client, blockHeight)
			if err != nil {
				return nil, err
			}
			blockHash = blk.Hash.Hex()
		} else {
			blockHash = *request.BlockIdentifier.Hash
			blk, err := cmn.GetBlockIdentifierByHash(s.client, *request.BlockIdentifier.Hash)
			if err != nil {
				return nil, err
			}
			blockHeight = blk.Height
		}
	}

	rpcRes, rpcErr := s.client.Call("theta.GetAccount", GetAccountArgs{
		Address: request.AccountIdentifier.Address,
		Height:  blockHeight,
	})

	if rpcRes != nil && rpcRes.Error != nil && rpcRes.Error.Code == -32000 {
		resp := types.AccountBalanceResponse{}
		resp.BlockIdentifier = &types.BlockIdentifier{Index: int64(blockHeight), Hash: blockHash}

		var thetaBalance types.Amount
		thetaBalance.Value = "0"
		thetaBalance.Currency = cmn.GetThetaCurrency()
		resp.Balances = append(resp.Balances, &thetaBalance)
		var tfuelBalance types.Amount
		tfuelBalance.Value = "0"
		tfuelBalance.Currency = cmn.GetTFuelCurrency()
		resp.Balances = append(resp.Balances, &tfuelBalance)

		return &resp, nil
	}

	parse := func(jsonBytes []byte) (interface{}, error) {
		account := GetAccountResult{}.Account
		err := json.Unmarshal(jsonBytes, &account)
		if err != nil {
			return nil, err
		}

		resp := types.AccountBalanceResponse{}
		resp.BlockIdentifier = &types.BlockIdentifier{Index: int64(blockHeight), Hash: blockHash}
		resp.Metadata = map[string]interface{}{"sequence_number": account.Sequence}

		var needTheta, needTFuel bool
		if request.Currencies != nil {
			for _, currency := range request.Currencies {
				if strings.EqualFold(currency.Symbol, ttypes.DenomThetaWei) {
					needTheta = true
				} else if strings.EqualFold(currency.Symbol, ttypes.DenomTFuelWei) {
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
			thetaBalance.Currency = cmn.GetThetaCurrency()
			resp.Balances = append(resp.Balances, &thetaBalance)
		}

		if needTFuel {
			var tfuelBalance types.Amount
			tfuelBalance.Value = account.Balance.TFuelWei.String()
			tfuelBalance.Currency = cmn.GetTFuelCurrency()
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
	if !strings.EqualFold(cmn.CfgRosettaModeOnline, viper.GetString(cmn.CfgRosettaMode)) {
		return nil, cmn.ErrUnavailableOffline
	}

	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	status, err := cmn.GetStatus(s.client)
	blockIdentifier := &types.BlockIdentifier{Index: int64(status.LatestFinalizedBlockHeight), Hash: status.LatestFinalizedBlockHash.String()}

	rpcRes, rpcErr := s.client.Call("theta.GetAccount", GetAccountArgs{
		Address: request.AccountIdentifier.Address,
	})

	if rpcRes != nil && rpcRes.Error != nil && rpcRes.Error.Code == -32000 {
		resp := types.AccountCoinsResponse{}
		resp.BlockIdentifier = blockIdentifier

		var thetaCoin types.Coin
		thetaCoin.CoinIdentifier = &types.CoinIdentifier{Identifier: ttypes.DenomThetaWei}
		var thetaBalance types.Amount
		thetaBalance.Value = "0"
		thetaBalance.Currency = cmn.GetThetaCurrency()
		thetaCoin.Amount = &thetaBalance
		resp.Coins = append(resp.Coins, &thetaCoin)

		var tfuelCoin types.Coin
		tfuelCoin.CoinIdentifier = &types.CoinIdentifier{Identifier: ttypes.DenomTFuelWei}
		var tfuelBalance types.Amount
		tfuelBalance.Value = "0"
		tfuelBalance.Currency = cmn.GetTFuelCurrency()
		tfuelCoin.Amount = &tfuelBalance
		resp.Coins = append(resp.Coins, &tfuelCoin)

		return &resp, nil
	}

	parse := func(jsonBytes []byte) (interface{}, error) {
		account := GetAccountResult{}.Account
		json.Unmarshal(jsonBytes, &account)

		resp := types.AccountCoinsResponse{}
		resp.BlockIdentifier = blockIdentifier
		resp.Metadata = map[string]interface{}{"sequence_number": account.Sequence}

		var needTheta, needTFuel bool
		if request.Currencies != nil {
			for _, currency := range request.Currencies {
				if strings.EqualFold(currency.Symbol, ttypes.DenomThetaWei) {
					needTheta = true
				} else if strings.EqualFold(currency.Symbol, ttypes.DenomTFuelWei) {
					needTFuel = true
				}
			}
		} else {
			needTheta = true
			needTFuel = true
		}

		if needTheta {
			var thetaCoin types.Coin
			thetaCoin.CoinIdentifier = &types.CoinIdentifier{Identifier: ttypes.DenomThetaWei}
			var thetaBalance types.Amount
			thetaBalance.Value = account.Balance.ThetaWei.String()
			thetaBalance.Currency = cmn.GetThetaCurrency()
			thetaCoin.Amount = &thetaBalance
			resp.Coins = append(resp.Coins, &thetaCoin)
		}

		if needTFuel {
			var tfuelCoin types.Coin
			tfuelCoin.CoinIdentifier = &types.CoinIdentifier{Identifier: ttypes.DenomTFuelWei}
			var tfuelBalance types.Amount
			tfuelBalance.Value = account.Balance.TFuelWei.String()
			tfuelBalance.Currency = cmn.GetTFuelCurrency()
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
