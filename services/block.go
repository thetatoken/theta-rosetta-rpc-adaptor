package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/spf13/viper"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	jrpc "github.com/ybbus/jsonrpc"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	ttypes "github.com/thetatoken/theta/ledger/types"
)

type blockAPIService struct {
	client       jrpc.RPCClient
	db           *cmn.LDBDatabase
	stakeService *cmn.StakeService
}

type GetBlockArgs struct {
	Hash common.Hash `json:"hash"`
}

type GetBlockByHeightArgs struct {
	Height common.JSONUint64 `json:"height"`
}

type GetBlockResult struct {
	*GetBlockResultInner
}

type GetTransactionArgs struct {
	Hash string `json:"hash"`
}

type GetTransactionResult struct {
	BlockHash      common.Hash                       `json:"block_hash"`
	BlockHeight    common.JSONUint64                 `json:"block_height"`
	Status         cmn.TxStatus                      `json:"status"`
	TxHash         common.Hash                       `json:"hash"`
	Type           byte                              `json:"type"`
	Tx             ttypes.Tx                         `json:"transaction"`
	Receipt        *blockchain.TxReceiptEntry        `json:"receipt"`
	BalanceChanges *blockchain.TxBalanceChangesEntry `json:"blance_changes"`
}

type GetBlockResultInner struct {
	ChainID            string                   `json:"chain_id"`
	Epoch              common.JSONUint64        `json:"epoch"`
	Height             common.JSONUint64        `json:"height"`
	Parent             common.Hash              `json:"parent"`
	TxHash             common.Hash              `json:"transactions_hash"`
	StateHash          common.Hash              `json:"state_hash"`
	Timestamp          *common.JSONBig          `json:"timestamp"`
	Proposer           common.Address           `json:"proposer"`
	HCC                core.CommitCertificate   `json:"hcc"`
	GuardianVotes      *core.AggregatedVotes    `json:"guardian_votes"`
	EliteEdgeNodeVotes *core.AggregatedEENVotes `json:"elite_edge_node_votes"`

	Children []common.Hash   `json:"children"`
	Status   cmn.BlockStatus `json:"status"` // cmn.BlockStatus has String(). Or core.Status?

	Hash common.Hash `json:"hash"`
	Txs  []cmn.Tx    `json:"transactions"`
}

// NewBlockAPIService creates a new instance of an AccountAPIService.
func NewBlockAPIService(client jrpc.RPCClient, db *cmn.LDBDatabase, stakeService *cmn.StakeService) server.BlockAPIServicer {
	return &blockAPIService{
		client:       client,
		db:           db,
		stakeService: stakeService,
	}
}

func (s *blockAPIService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if !strings.EqualFold(cmn.CfgRosettaModeOnline, viper.GetString(cmn.CfgRosettaMode)) {
		return nil, cmn.ErrUnavailableOffline
	}

	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	var rpcRes *jrpc.RPCResponse
	var rpcErr error

	if request.BlockIdentifier.Index != nil {
		rpcRes, rpcErr = s.client.Call("theta.GetBlockByHeight", GetBlockByHeightArgs{
			Height: common.JSONUint64(*request.BlockIdentifier.Index),
		})
	} else if request.BlockIdentifier.Hash != nil {
		rpcRes, rpcErr = s.client.Call("theta.GetBlock", GetBlockArgs{
			Hash: common.HexToHash(*request.BlockIdentifier.Hash),
		})
	}

	if rpcErr != nil {
		return nil, cmn.ErrUnableToGetBlk
	}

	parse := func(jsonBytes []byte) (interface{}, error) {
		tblock := GetBlockResult{}
		json.Unmarshal(jsonBytes, &tblock)

		if tblock.GetBlockResultInner == nil {
			return nil, fmt.Errorf(cmn.ErrUnableToGetBlk.Message)
		}

		block := types.Block{}
		block.BlockIdentifier = &types.BlockIdentifier{Index: int64(tblock.Height), Hash: tblock.Hash.Hex()}
		var parentHeight int64
		if tblock.Height == 0 {
			parentHeight = 0
		} else {
			parentHeight = int64(tblock.Height - 1)
		}
		block.ParentBlockIdentifier = &types.BlockIdentifier{Index: parentHeight, Hash: tblock.Parent.Hex()}
		block.Timestamp = tblock.Timestamp.ToInt().Int64() * 1000
		block.Metadata = map[string]interface{}{
			"status":            tblock.Status.String(),
			"transactions_hash": tblock.TxHash,
			"state_hash":        tblock.StateHash.Hex(),
			"proposer":          tblock.Proposer.Hex(),
		} //TODO: anything else needed?

		txs := make([]*types.Transaction, 0)
		status := tblock.Status.String()
		var objMap map[string]json.RawMessage
		json.Unmarshal(jsonBytes, &objMap)
		if objMap["transactions"] != nil {
			var txMaps []map[string]json.RawMessage
			json.Unmarshal(objMap["transactions"], &txMaps)
			for i, txMap := range txMaps {
				var gasUsed uint64
				if tblock.Txs[i].Receipt != nil {
					gasUsed = tblock.Txs[i].Receipt.GasUsed
				}

				tx := cmn.ParseTx(tblock.Txs[i].Type, txMap["raw"], tblock.Txs[i].Hash, &status, gasUsed, tblock.Txs[i].BalanceChanges, s.db, s.stakeService, tblock.Height)
				txs = append(txs, &tx)
			}
		}

		// check if there's any stakes that need to be returned at this height
		returnStakeTxMap := cmn.ReturnStakeTxMap{}
		kvstore := cmn.NewKVStore(s.db)
		if kvstore.Get(new(big.Int).SetUint64(uint64(tblock.Height)).Bytes(), &returnStakeTxMap) == nil {
			for _, tx := range returnStakeTxMap.ReturnStakeMap {
				transaction := types.Transaction{
					TransactionIdentifier: &types.TransactionIdentifier{Hash: tblock.Hash.Hex()},
				}
				transaction.Metadata, transaction.Operations = cmn.ParseReturnStakeTx(tx.Tx, &status, cmn.WithdrawStakeTx)
				txs = append(txs, &transaction)
			}
		}

		block.Transactions = txs

		resp := types.BlockResponse{
			Block: &block,
		}
		return resp, nil
	}

	res, err := cmn.HandleThetaRPCResponse(rpcRes, rpcErr, parse)
	if err != nil {
		return nil, cmn.ErrUnableToGetBlk
	}

	ret, _ := res.(types.BlockResponse)
	return &ret, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *blockAPIService) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	if !strings.EqualFold(cmn.CfgRosettaModeOnline, viper.GetString(cmn.CfgRosettaMode)) {
		return nil, cmn.ErrUnavailableOffline
	}

	if err := cmn.ValidateNetworkIdentifier(ctx, request.NetworkIdentifier); err != nil {
		return nil, err
	}

	rpcRes, rpcErr := s.client.Call("theta.GetTransaction", GetTransactionArgs{
		Hash: request.TransactionIdentifier.Hash,
	})

	parse := func(jsonBytes []byte) (interface{}, error) {
		txResult := GetTransactionResult{}
		json.Unmarshal(jsonBytes, &txResult)

		resp := types.BlockTransactionResponse{}

		var objMap map[string]json.RawMessage
		json.Unmarshal(jsonBytes, &objMap)
		if objMap["transaction"] != nil {
			var rawTx json.RawMessage
			json.Unmarshal(objMap["transaction"], &rawTx)

			var gasUsed uint64
			if txResult.Receipt != nil {
				gasUsed = txResult.Receipt.GasUsed
			}

			status := string(txResult.Status)
			if "not_found" != status {
				tx := cmn.ParseTx(cmn.TxType(txResult.Type), rawTx, txResult.TxHash, &status, gasUsed, txResult.BalanceChanges, nil, nil, 0)
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

	ret, _ := res.(types.BlockTransactionResponse)
	return &ret, nil
}
