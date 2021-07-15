package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/spf13/viper"

	jrpc "github.com/ybbus/jsonrpc"

	cmn "github.com/thetatoken/theta-rosetta-rpc-adaptor/common"
	"github.com/thetatoken/theta/version"
)

type networkAPIService struct {
	client jrpc.RPCClient
}

// NewAccountAPIService creates a new instance of an AccountAPIService.
func NewNetworkAPIService(client jrpc.RPCClient) server.NetworkAPIServicer {
	return &networkAPIService{
		client: client,
	}
}

// NetworkList implements the /network/list endpoint.
func (s *networkAPIService) NetworkList(
	ctx context.Context,
	request *types.MetadataRequest,
) (*types.NetworkListResponse, *types.Error) {
	return &types.NetworkListResponse{
		NetworkIdentifiers: []*types.NetworkIdentifier{{
			Blockchain: "theta",
			Network:    cmn.GetChainId(),
		},
		},
	}, nil
}

// NetworkStatus implements the /network/status endpoint.
func (s *networkAPIService) NetworkStatus(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkStatusResponse, *types.Error) {
	// terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier)
	// if terr != nil {
	// 	return nil, terr
	// }

	status, err := cmn.GetStatus(s.client)
	if err != nil {
		return nil, cmn.ErrUnableToGetNodeStatus
	}

	lastFinalized := int64(status.LatestFinalizedBlockHeight)
	currHeight := int64(status.CurrentHeight)
	synced := !status.Syncing

	skipEdgeNode := true
	if request.Metadata != nil {
		if val, ok := request.Metadata["skip_edge_node"]; ok {
			skipEdgeNode = val.(bool)
		}
	}

	peers, err := GetPeers(s.client, skipEdgeNode)
	if err != nil {
		return nil, cmn.ErrUnableToGetNodeStatus
	}

	peerList := make([]*types.Peer, 0)
	for _, peerId := range peers.Peers {
		peer := types.Peer{PeerID: peerId}
		peerList = append(peerList, &peer)
	}

	resp := &types.NetworkStatusResponse{
		CurrentBlockIdentifier: &types.BlockIdentifier{Index: lastFinalized, Hash: status.LatestFinalizedBlockHash.Hex()},
		CurrentBlockTimestamp:  status.LatestFinalizedBlockTime.ToInt().Int64(),
		GenesisBlockIdentifier: &types.BlockIdentifier{Index: 0, Hash: status.GenesisBlockHash.Hex()},
		SyncStatus:             &types.SyncStatus{CurrentIndex: &lastFinalized, TargetIndex: &currHeight, Synced: &synced},
		Peers:                  peerList,
	}

	return resp, nil
}

// NetworkOptions implements the /network/options endpoint.
func (s *networkAPIService) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
	// 	terr := ValidateNetworkIdentifier(ctx, s.client, request.NetworkIdentifier)
	// 	if terr != nil {
	// 		return nil, terr
	// 	}

	return &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion: viper.GetString(cmn.CfgRosettaVersion),
			NodeVersion:    version.Version,
		},
		Allow: &types.Allow{
			OperationStatuses: []*types.OperationStatus{
				{
					Status:     cmn.StatusSuccess, //TODO ?
					Successful: true,
				},
				{
					Status:     cmn.StatusFail, //TODO ?
					Successful: false,
				},
			},
			OperationTypes:          cmn.TxOpTypes(),
			Errors:                  cmn.ErrorList,
			HistoricalBalanceLookup: true,
			MempoolCoins:            true, // Any Rosetta implementation that can update an AccountIdentifier's unspent coins based on the
			// contents of the mempool should populate this field as true. If false, requests to
			// `/account/coins` that set `include_mempool` as true will be automatically rejected
		},
	}, nil
}

type GetPeersArgs struct {
	SkipEdgeNode bool `json:"skip_edge_node"`
}

type GetPeersResult struct {
	Peers []string `json:"peers"`
}

func GetPeers(client jrpc.RPCClient, skipEdgeNode bool) (*GetPeersResult, error) {
	rpcRes, rpcErr := client.Call("theta.GetPeers", GetPeersArgs{SkipEdgeNode: skipEdgeNode})
	if rpcErr != nil {
		return nil, rpcErr
	}
	jsonBytes, err := json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))
	}
	trpcResult := GetPeersResult{}
	json.Unmarshal(jsonBytes, &trpcResult)

	return &trpcResult, nil
}
