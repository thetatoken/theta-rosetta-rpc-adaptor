package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	rpcc "github.com/ybbus/jsonrpc"

	"github.com/coinbase/rosetta-sdk-go/types"
	log "github.com/sirupsen/logrus"
	cmn "github.com/thetatoken/theta/common"
	ttypes "github.com/thetatoken/theta/ledger/types"
	jrpc "github.com/ybbus/jsonrpc"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "utils"})

// ------------------------------ Chain ID -----------------------------------
var chainId string

func GetChainId() string {
	return chainId
}

func SetChainId(chid string) {
	chainId = chid
}

// ------------------------------ Theta RPC -----------------------------------

func GetThetaRPCEndpoint() string {
	thetaRPCEndpoint := viper.GetString(CfgThetaRPCEndpoint)
	return thetaRPCEndpoint
}

func HandleThetaRPCResponse(rpcRes *rpcc.RPCResponse, rpcErr error, parse func(jsonBytes []byte) (interface{}, error)) (result interface{}, err error) {
	if rpcErr != nil {
		return nil, fmt.Errorf("failed to get theta RPC response: %v", rpcErr)
	}
	if rpcRes.Error != nil {
		return nil, fmt.Errorf("theta RPC returns an error: %v", rpcRes.Error)
	}

	var jsonBytes []byte
	jsonBytes, err = json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))
	}

	result, err = parse(jsonBytes)
	return
}

// ------------------------------ Validate Network Identifier -----------------------------------

func ValidateNetworkIdentifier(ctx context.Context, ni *types.NetworkIdentifier) *types.Error {
	if ni != nil {
		if !strings.EqualFold(ni.Blockchain, Theta) {
			return ErrInvalidBlockchain
		}
		if ni.SubNetworkIdentifier != nil {
			return ErrInvalidSubnetwork
		}
		if !strings.EqualFold(ni.Network, GetChainId()) {
			return ErrInvalidNetwork
		}
	} else {
		return ErrMissingNID
	}
	return nil
}

// ------------------------------ GetStatus -----------------------------------

type GetStatusArgs struct{}

type GetStatusResult struct {
	Address                    string         `json:"address"`
	ChainID                    string         `json:"chain_id"`
	PeerID                     string         `json:"peer_id"`
	LatestFinalizedBlockHash   cmn.Hash       `json:"latest_finalized_block_hash"`
	LatestFinalizedBlockHeight cmn.JSONUint64 `json:"latest_finalized_block_height"`
	LatestFinalizedBlockTime   *cmn.JSONBig   `json:"latest_finalized_block_time"`
	LatestFinalizedBlockEpoch  cmn.JSONUint64 `json:"latest_finalized_block_epoch"`
	CurrentEpoch               cmn.JSONUint64 `json:"current_epoch"`
	CurrentHeight              cmn.JSONUint64 `json:"current_height"`
	CurrentTime                *cmn.JSONBig   `json:"current_time"`
	Syncing                    bool           `json:"syncing"`
	GenesisBlockHash           cmn.Hash       `json:"genesis_block_hash"`
}

func GetStatus(client jrpc.RPCClient) (*GetStatusResult, error) {
	rpcRes, rpcErr := client.Call("theta.GetStatus", GetStatusArgs{})
	if rpcErr != nil {
		return nil, rpcErr
	}
	jsonBytes, err := json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))
	}
	trpcResult := GetStatusResult{}
	json.Unmarshal(jsonBytes, &trpcResult)

	return &trpcResult, nil
}

// ------------------------------ Tx -----------------------------------

func ParseTx(tx Tx, txMap map[string]json.RawMessage, status *string) types.Transaction { //[]*types.Operation {
	switch tx.Type {
	case Coinbase: //*ttypes.CoinbaseTx:
		// coinbaseTx := tx.Tx.(*ttypes.CoinbaseTx)
		coinbaseTx := ttypes.CoinbaseTx{}
		json.Unmarshal(txMap["raw"], &coinbaseTx)

		transaction := types.Transaction{
			TransactionIdentifier: &types.TransactionIdentifier{Hash: tx.Hash.Hex()},
			Metadata:              map[string]interface{}{"block_height": coinbaseTx.BlockHeight},
		}

		transaction.Operations = []*types.Operation{
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 0},
				Type:                CoinbaseProposer.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: coinbaseTx.Proposer.Address.String()},
				Amount:              &types.Amount{Value: coinbaseTx.Proposer.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
				Metadata:            map[string]interface{}{"sequence": coinbaseTx.Proposer.Sequence, "signature": coinbaseTx.Proposer.Signature},
			},
		}

		for i, output := range coinbaseTx.Outputs {
			outputOp := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: int64(i) + 1},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: 0}},
				Type:                CoinbaseOutput.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: output.Address.String()},
				Amount:              &types.Amount{Value: output.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			}
			transaction.Operations = append(transaction.Operations, outputOp)
		}

		return transaction

		// case *ttypes.SlashTx:
		// case *ttypes.SendTx:
		// case *ttypes.ReserveFundTx:
		// case *ttypes.ReleaseFundTx:
		// case *ttypes.ServicePaymentTx:
		// case *ttypes.SplitRuleTx:
		// case *ttypes.SmartContractTx:
		// case *ttypes.DepositStakeTx:
		// case *ttypes.WithdrawStakeTx:
		// case *ttypes.DepositStakeTxV2:
		// case *ttypes.StakeRewardDistributionTx:
	}

	return types.Transaction{}
}
