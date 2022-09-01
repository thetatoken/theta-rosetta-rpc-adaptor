package common

import (
	"encoding/json"
	"fmt"
	"math/big"

	cmn "github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	ttypes "github.com/thetatoken/theta/ledger/types"
	jrpc "github.com/ybbus/jsonrpc"
)

type GetStakeByHeightArgs struct {
	Height cmn.JSONUint64 `json:"height"`
}

type GetVcpResult struct {
	BlockHashVcpPairs []BlockHashVcpPair
}

type BlockHashVcpPair struct {
	BlockHash  cmn.Hash
	Vcp        *core.ValidatorCandidatePool
	HeightList *ttypes.HeightList
}

type GetGcpResult struct {
	BlockHashGcpPairs []BlockHashGcpPair
}

type BlockHashGcpPair struct {
	BlockHash cmn.Hash
	Gcp       *core.GuardianCandidatePool
}

type GetEenpResult struct {
	BlockHashEenpPairs []BlockHashEenpPair
}

type BlockHashEenpPair struct {
	BlockHash cmn.Hash
	EENs      []*core.EliteEdgeNode
}

type GetEenpStakeByHeightArgs struct {
	Height        cmn.JSONUint64 `json:"height"`
	Source        cmn.Address    `json:"source"`
	Holder        cmn.Address    `json:"holder"`
	WithdrawnOnly bool           `json:"withdrawn_only"`
}

type GetEenpStakeResult struct {
	Stake core.Stake `json:"stake"`
}

type StakeService struct {
	client jrpc.RPCClient
	db     *LDBDatabase
}

const StakeReturnPrefix = "stake_return"

func NewStakeService(client jrpc.RPCClient, db *LDBDatabase) *StakeService {
	return &StakeService{
		client: client,
		db:     db,
	}
}

func (ss *StakeService) GenStakesForSnapshot() error {
	returnStakeTxsMap := make(map[uint64]ReturnStakeTxs)

	status, err := GetStatus(ss.client)
	if err != nil {
		return nil
	}
	snapshotHeight := status.SnapshotBlockHeight

	// VCP
	rpcRes, rpcErr := ss.client.Call("theta.GetVcpByHeight", GetStakeByHeightArgs{
		Height: snapshotHeight,
	})

	if rpcErr != nil {
		return rpcErr
	}
	if rpcRes != nil && rpcRes.Error != nil {
		return rpcRes.Error
	}

	jsonBytes, err := json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))
	}

	vcpResult := GetVcpResult{}
	json.Unmarshal(jsonBytes, &vcpResult)
	if len(vcpResult.BlockHashVcpPairs) > 0 {
		for _, candidate := range vcpResult.BlockHashVcpPairs[0].Vcp.SortedCandidates {
			blockHash := vcpResult.BlockHashVcpPairs[0].BlockHash.Hex()
			for i, stake := range candidate.Stakes {
				if stake.Withdrawn {
					withdrawStakeTx := ttypes.WithdrawStakeTx{
						Source: ttypes.TxInput{
							Address: stake.Source,
							Coins:   ttypes.Coins{ThetaWei: stake.Amount, TFuelWei: big.NewInt(0)},
						},
						Holder: ttypes.TxOutput{
							Address: stake.Holder,
						},
					}
					hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("vcp_%s_%s_%d", StakeReturnPrefix, blockHash, i)))
					returnStakeTx := &ReturnStakeTx{
						Hash: hash.Hex(),
						Tx:   withdrawStakeTx,
					}
					if returnStakeTxs, ok := returnStakeTxsMap[stake.ReturnHeight]; ok {
						returnStakeTxs.ReturnStakes = append(returnStakeTxs.ReturnStakes, returnStakeTx)
						returnStakeTxsMap[stake.ReturnHeight] = returnStakeTxs
					} else {
						returnStakeTxsMap[stake.ReturnHeight] = ReturnStakeTxs{[]*ReturnStakeTx{returnStakeTx}}
					}
				}
			}
		}
	}

	// GCP
	rpcRes, rpcErr = ss.client.Call("theta.GetGcpByHeight", GetStakeByHeightArgs{
		Height: snapshotHeight,
	})

	if rpcErr != nil {
		return rpcErr
	}
	if rpcRes != nil && rpcRes.Error != nil {
		return rpcRes.Error
	}

	jsonBytes, err = json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))
	}

	gcpResult := GetGcpResult{}
	json.Unmarshal(jsonBytes, &gcpResult)
	if len(gcpResult.BlockHashGcpPairs) > 0 {
		for _, guardian := range gcpResult.BlockHashGcpPairs[0].Gcp.SortedGuardians {
			blockHash := gcpResult.BlockHashGcpPairs[0].BlockHash.Hex()
			for i, stake := range guardian.Stakes {
				if stake.Withdrawn {
					withdrawStakeTx := ttypes.WithdrawStakeTx{
						Source: ttypes.TxInput{
							Address: stake.Source,
							Coins:   ttypes.Coins{ThetaWei: stake.Amount, TFuelWei: big.NewInt(0)},
						},
						Holder: ttypes.TxOutput{
							Address: stake.Holder,
						},
					}
					hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("gcp_%s_%s_%d", StakeReturnPrefix, blockHash, i)))
					returnStakeTx := &ReturnStakeTx{
						Hash: hash.Hex(),
						Tx:   withdrawStakeTx,
					}
					if returnStakeTxs, ok := returnStakeTxsMap[stake.ReturnHeight]; ok {
						returnStakeTxs.ReturnStakes = append(returnStakeTxs.ReturnStakes, returnStakeTx)
					} else {
						returnStakeTxsMap[stake.ReturnHeight] = ReturnStakeTxs{[]*ReturnStakeTx{returnStakeTx}}
					}
				}
			}
		}
	}

	// EENP
	rpcRes, rpcErr = ss.client.Call("theta.GetEenpByHeight", GetStakeByHeightArgs{
		Height: snapshotHeight,
	})

	if rpcErr != nil {
		return rpcErr
	}
	if rpcRes != nil && rpcRes.Error != nil {
		return rpcRes.Error
	}

	jsonBytes, err = json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))
	}

	eenpResult := GetEenpResult{}
	json.Unmarshal(jsonBytes, &eenpResult)
	if len(eenpResult.BlockHashEenpPairs) > 0 {
		for _, een := range eenpResult.BlockHashEenpPairs[0].EENs {
			blockHash := eenpResult.BlockHashEenpPairs[0].BlockHash.Hex()
			for i, stake := range een.Stakes {
				if stake.Withdrawn {
					withdrawStakeTx := ttypes.WithdrawStakeTx{
						Source: ttypes.TxInput{
							Address: stake.Source,
							Coins:   ttypes.Coins{ThetaWei: big.NewInt(0), TFuelWei: stake.Amount},
						},
						Holder: ttypes.TxOutput{
							Address: stake.Holder,
						},
					}
					hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("eenp_%s_%s_%d", StakeReturnPrefix, blockHash, i)))
					returnStakeTx := &ReturnStakeTx{
						Hash: hash.Hex(),
						Tx:   withdrawStakeTx,
					}
					if returnStakeTxs, ok := returnStakeTxsMap[stake.ReturnHeight]; ok {
						returnStakeTxs.ReturnStakes = append(returnStakeTxs.ReturnStakes, returnStakeTx)
					} else {
						returnStakeTxsMap[stake.ReturnHeight] = ReturnStakeTxs{[]*ReturnStakeTx{returnStakeTx}}
					}
				}
			}
		}
	}

	// store in db
	kvstore := NewKVStore(ss.db)
	for height, returnStakeTxs := range returnStakeTxsMap {
		heightBytes := new(big.Int).SetUint64(uint64(height + 1)).Bytes() // actual return height is off-by-one
		kvstore.Put(heightBytes, returnStakeTxs)
	}

	return nil
}

func (ss *StakeService) GetStakeForTx(withdrawStakeTx ttypes.WithdrawStakeTx, blockHeight cmn.JSONUint64) (*core.Stake, error) {
	var args interface{}
	rpcMethod := "theta."
	switch withdrawStakeTx.Purpose {
	case core.StakeForValidator:
		rpcMethod += "GetVcpByHeight"
		args = GetStakeByHeightArgs{Height: blockHeight}
	case core.StakeForGuardian:
		rpcMethod += "GetGcpByHeight"
		args = GetStakeByHeightArgs{Height: blockHeight}
	case core.StakeForEliteEdgeNode:
		rpcMethod += "GetEenpStakeByHeight"
		args = GetEenpStakeByHeightArgs{
			Height:        blockHeight,
			Source:        withdrawStakeTx.Source.Address,
			Holder:        withdrawStakeTx.Holder.Address,
			WithdrawnOnly: true,
		}
	}

	rpcRes, rpcErr := ss.client.Call(rpcMethod, args)

	if rpcErr != nil {
		return nil, rpcErr
	}
	if rpcRes != nil && rpcRes.Error != nil {
		return nil, rpcRes.Error
	}

	jsonBytes, err := json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))
	}

	switch withdrawStakeTx.Purpose {
	case core.StakeForValidator:
		vcpResult := GetVcpResult{}
		json.Unmarshal(jsonBytes, &vcpResult)
		if len(vcpResult.BlockHashVcpPairs) > 0 {
			for _, candidate := range vcpResult.BlockHashVcpPairs[0].Vcp.SortedCandidates {
				if candidate.Holder == withdrawStakeTx.Holder.Address {
					for _, stake := range candidate.Stakes {
						if stake.Source == withdrawStakeTx.Source.Address {
							return stake, nil
						}
					}
				}
			}
		}
	case core.StakeForGuardian:
		gcpResult := GetGcpResult{}
		json.Unmarshal(jsonBytes, &gcpResult)
		if len(gcpResult.BlockHashGcpPairs) > 0 {
			for _, guardian := range gcpResult.BlockHashGcpPairs[0].Gcp.SortedGuardians {
				if guardian.Holder == withdrawStakeTx.Holder.Address {
					for _, stake := range guardian.Stakes {
						if stake.Source == withdrawStakeTx.Source.Address {
							return stake, nil
						}
					}
				}
			}
		}
	case core.StakeForEliteEdgeNode:
		eenpStakeResult := GetEenpStakeResult{}
		json.Unmarshal(jsonBytes, &eenpStakeResult)
		return &eenpStakeResult.Stake, nil
	}
	return nil, nil
}
