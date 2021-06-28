package services

import (
	"context"
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	jrpc "github.com/ybbus/jsonrpc"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
)

type blockAPIService struct {
	client jrpc.RPCClient
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
func NewBlockAPIService(client jrpc.RPCClient) server.BlockAPIServicer {
	return &blockAPIService{
		client: client,
	}
}

func (s *blockAPIService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	// terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier)
	// if terr != nil {
	// 	return nil, terr
	// }

	var rpcRes *jrpc.RPCResponse
	var rpcErr error

	if request.BlockIdentifier.Hash != nil {
		rpcRes, rpcErr = s.client.Call("theta.GetBlock", GetBlockArgs{
			Hash: common.HexToHash(*request.BlockIdentifier.Hash),
		})
	} else if request.BlockIdentifier.Index != nil {
		rpcRes, rpcErr = s.client.Call("theta.GetBlockByHeight", GetBlockByHeightArgs{
			Height: common.JSONUint64(*request.BlockIdentifier.Index),
		})
	} else {
		return nil, cmn.ErrMissingBlockHashOrHeight
	}

	if rpcErr != nil {
		return nil, cmn.ErrUnableToGetBlk
	}

	parse := func(jsonBytes []byte) (interface{}, error) {
		tblock := GetBlockResult{}
		json.Unmarshal(jsonBytes, &tblock)

		block := types.Block{}
		block.BlockIdentifier = &types.BlockIdentifier{Index: int64(tblock.Height), Hash: tblock.Hash.Hex()}
		block.ParentBlockIdentifier = &types.BlockIdentifier{Index: int64(tblock.Height - 1), Hash: tblock.Parent.Hex()}
		block.Timestamp = tblock.Timestamp.ToInt().Int64()
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
				tx := cmn.ParseTx(tblock.Txs[i], txMap, &status)
				txs = append(txs, &tx)
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
	// terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier)
	// if terr != nil {
	// 	return nil, terr
	// }

	// transaction, err := s.client.GetBlockTransaction(ctx, request.TransactionIdentifier.Hash)
	// if err != nil {
	// 	return nil, ErrUnableToGetBlkTx
	// }

	// return &types.BlockTransactionResponse{
	// 	Transaction: transaction,
	// }, nil

	return nil, nil
}
