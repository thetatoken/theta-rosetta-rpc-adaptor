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

func ParseTx(txType TxType, rawTx json.RawMessage, txHash cmn.Hash, status *string) types.Transaction {
	transaction := types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: txHash.String()},
	}

	switch txType {
	case Coinbase:
		coinbaseTx := ttypes.CoinbaseTx{}
		json.Unmarshal(rawTx, &coinbaseTx)

		transaction.Metadata = map[string]interface{}{"block_height": coinbaseTx.BlockHeight}

		transaction.Operations = []*types.Operation{
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 0},
				Type:                CoinbaseTxProposer.String(),
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
				Type:                CoinbaseTxOutput.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: output.Address.String()},
				Amount:              &types.Amount{Value: output.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			}
			transaction.Operations = append(transaction.Operations, outputOp)
		}

	case Slash:
		slashTx := ttypes.SlashTx{}
		json.Unmarshal(rawTx, &slashTx)

		transaction.Metadata = map[string]interface{}{
			"slashed_address":  slashTx.SlashedAddress,
			"reserve_sequence": slashTx.ReserveSequence,
			"slash_proof":      slashTx.SlashProof, //TODO: need to convert to hex string?
		}

		transaction.Operations = []*types.Operation{
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 0},
				Type:                SlashTxProposer.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: slashTx.Proposer.Address.String()},
				Amount:              &types.Amount{Value: slashTx.Proposer.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
				Metadata:            map[string]interface{}{"sequence": slashTx.Proposer.Sequence, "signature": slashTx.Proposer.Signature},
			},
		}

	case Send:
		sendTx := ttypes.SendTx{}
		json.Unmarshal(rawTx, &sendTx)

		transaction.Metadata = map[string]interface{}{
			"fee": sendTx.Fee,
		}

		var i int64
		for _, input := range sendTx.Inputs {
			if input.Coins.ThetaWei != nil {
				inputOp := &types.Operation{
					OperationIdentifier: &types.OperationIdentifier{Index: i},
					Type:                SendTxInput.String(),
					Status:              status, // same as block status
					Account:             &types.AccountIdentifier{Address: input.Address.String()},
					Amount:              &types.Amount{Value: input.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
					Metadata:            map[string]interface{}{"sequence": input.Sequence, "signature": input.Signature},
				}
				if i > 0 {
					inputOp.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
				}
				transaction.Operations = append(transaction.Operations, inputOp)
				i++
			}
			if input.Coins.TFuelWei != nil {
				inputOp := &types.Operation{
					OperationIdentifier: &types.OperationIdentifier{Index: i},
					Type:                SendTxInput.String(),
					Status:              status, // same as block status
					Account:             &types.AccountIdentifier{Address: input.Address.String()},
					Amount:              &types.Amount{Value: input.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
					Metadata:            map[string]interface{}{"sequence": input.Sequence, "signature": input.Signature},
				}
				if i > 0 {
					inputOp.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
				}
				transaction.Operations = append(transaction.Operations, inputOp)
				i++
			}
		}

		for _, output := range sendTx.Outputs {
			if output.Coins.ThetaWei != nil {
				outputOp := &types.Operation{
					OperationIdentifier: &types.OperationIdentifier{Index: i},
					RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
					Type:                SendTxOutput.String(),
					Status:              status, // same as block status
					Account:             &types.AccountIdentifier{Address: output.Address.String()},
					Amount:              &types.Amount{Value: output.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
				}
				transaction.Operations = append(transaction.Operations, outputOp)
				i++
			}
			if output.Coins.TFuelWei != nil {
				outputOp := &types.Operation{
					OperationIdentifier: &types.OperationIdentifier{Index: i},
					RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
					Type:                SendTxOutput.String(),
					Status:              status, // same as block status
					Account:             &types.AccountIdentifier{Address: output.Address.String()},
					Amount:              &types.Amount{Value: output.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
				}
				transaction.Operations = append(transaction.Operations, outputOp)
				i++
			}
		}

	case ReserveFund:
		reserveFundTx := ttypes.ReserveFundTx{}
		json.Unmarshal(rawTx, &reserveFundTx)

		transaction.Metadata = map[string]interface{}{
			"fee":          reserveFundTx.Fee,
			"collateral":   reserveFundTx.Collateral,
			"resource_ids": reserveFundTx.ResourceIDs,
			"duration":     reserveFundTx.Duration,
		}

		transaction.Operations = []*types.Operation{
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 0},
				Type:                ReserveFundTxSource.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: reserveFundTx.Source.Address.String()},
				Amount:              &types.Amount{Value: reserveFundTx.Source.Coins.ThetaWei.String(), Currency: GetThetaCurrency()}, //TODO: theta only?
				Metadata:            map[string]interface{}{"sequence": reserveFundTx.Source.Sequence, "signature": reserveFundTx.Source.Signature},
			},
		}

	case ReleaseFund:
		releaseFundTx := ttypes.ReleaseFundTx{}
		json.Unmarshal(rawTx, &releaseFundTx)

		transaction.Metadata = map[string]interface{}{
			"fee":              releaseFundTx.Fee,
			"reserve_sequence": releaseFundTx.ReserveSequence,
		}

		transaction.Operations = []*types.Operation{
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 0},
				Type:                ReleaseFundTxSource.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: releaseFundTx.Source.Address.String()},
				Amount:              &types.Amount{Value: releaseFundTx.Source.Coins.ThetaWei.String(), Currency: GetThetaCurrency()}, //TODO: theta only?
				Metadata:            map[string]interface{}{"sequence": releaseFundTx.Source.Sequence, "signature": releaseFundTx.Source.Signature},
			},
		}

	case ServicePayment:
		servicePaymentTx := ttypes.ServicePaymentTx{}
		json.Unmarshal(rawTx, &servicePaymentTx)

		transaction.Metadata = map[string]interface{}{
			"fee":              servicePaymentTx.Fee,
			"payment_sequence": servicePaymentTx.PaymentSequence,
			"reserve_sequence": servicePaymentTx.ReserveSequence,
			"resource_id":      servicePaymentTx.ResourceID,
		}

		transaction.Operations = []*types.Operation{
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 0},
				Type:                ServicePaymentTxSource.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: servicePaymentTx.Source.Address.String()},
				Amount:              &types.Amount{Value: servicePaymentTx.Source.Coins.ThetaWei.String(), Currency: GetThetaCurrency()}, //TODO: theta only?
				Metadata:            map[string]interface{}{"sequence": servicePaymentTx.Source.Sequence, "signature": servicePaymentTx.Source.Signature},
			},
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 1},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: 0}},
				Type:                ServicePaymentTxTarget.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: servicePaymentTx.Target.Address.String()},
				Amount:              &types.Amount{Value: servicePaymentTx.Target.Coins.ThetaWei.String(), Currency: GetThetaCurrency()}, //TODO: theta only?
				Metadata:            map[string]interface{}{"sequence": servicePaymentTx.Target.Sequence, "signature": servicePaymentTx.Target.Signature},
			},
		}

	case SplitRule:
		splitRuleTx := ttypes.SplitRuleTx{}
		json.Unmarshal(rawTx, &splitRuleTx)

		transaction.Metadata = map[string]interface{}{
			"fee":         splitRuleTx.Fee,
			"resource_id": splitRuleTx.ResourceID,
			"splits":      splitRuleTx.Splits, //TODO ?
			"duration":    splitRuleTx.Duration,
		}

		transaction.Operations = []*types.Operation{
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 0},
				Type:                SplitRuleTxInitiator.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: splitRuleTx.Initiator.Address.String()},
				Amount:              &types.Amount{Value: splitRuleTx.Initiator.Coins.ThetaWei.String(), Currency: GetThetaCurrency()}, //TODO: theta only?
				Metadata:            map[string]interface{}{"sequence": splitRuleTx.Initiator.Sequence, "signature": splitRuleTx.Initiator.Signature},
			},
		}

	case SmartContract:
		smartContractTx := ttypes.SmartContractTx{}
		json.Unmarshal(rawTx, &smartContractTx)

		transaction.Metadata = map[string]interface{}{
			"GasLimit": smartContractTx.GasLimit,
			"GasPrice": smartContractTx.GasPrice,
			"data":     smartContractTx.Data,
		}

		transaction.Operations = []*types.Operation{
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 0},
				Type:                SmartContractTxFrom.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: smartContractTx.From.Address.String()},
				Amount:              &types.Amount{Value: smartContractTx.From.Coins.ThetaWei.String(), Currency: GetThetaCurrency()}, //TODO: theta only?
				Metadata:            map[string]interface{}{"sequence": smartContractTx.From.Sequence, "signature": smartContractTx.From.Signature},
			},
			&types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: 1},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: 0}},
				Type:                SmartContractTxTo.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: smartContractTx.To.Address.String()},
				Amount:              &types.Amount{Value: smartContractTx.To.Coins.ThetaWei.String(), Currency: GetThetaCurrency()}, //TODO: theta only?
			},
		}

	case DepositStake, DepositStakeV2:
		depositStakeTx := ttypes.DepositStakeTx{}
		json.Unmarshal(rawTx, &depositStakeTx)

		transaction.Metadata = map[string]interface{}{
			"fee":     depositStakeTx.Fee,
			"purpose": depositStakeTx.Purpose,
		}

		var i int64
		transaction.Operations = []*types.Operation{}
		if depositStakeTx.Source.Coins.ThetaWei != nil {
			thetaSource := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				Type:                DepositStakeTxSource.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: depositStakeTx.Source.Address.String()},
				Amount:              &types.Amount{Value: depositStakeTx.Source.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
				Metadata:            map[string]interface{}{"sequence": depositStakeTx.Source.Sequence, "signature": depositStakeTx.Source.Signature},
			}
			transaction.Operations = append(transaction.Operations, thetaSource)
			i++
		}
		if depositStakeTx.Source.Coins.TFuelWei != nil {
			tfuelSource := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				Type:                DepositStakeTxSource.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: depositStakeTx.Source.Address.String()},
				Amount:              &types.Amount{Value: depositStakeTx.Source.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
				Metadata:            map[string]interface{}{"sequence": depositStakeTx.Source.Sequence, "signature": depositStakeTx.Source.Signature},
			}
			if i > 0 {
				tfuelSource.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
			}
			transaction.Operations = append(transaction.Operations, tfuelSource)
			i++
		}
		if depositStakeTx.Holder.Coins.ThetaWei != nil {
			thetaHolder := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
				Type:                DepositStakeTxHolder.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: depositStakeTx.Holder.Address.String()},
				Amount:              &types.Amount{Value: depositStakeTx.Holder.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			}
			transaction.Operations = append(transaction.Operations, thetaHolder)
			i++
		}
		if depositStakeTx.Holder.Coins.TFuelWei != nil {
			tfuelHolder := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
				Type:                DepositStakeTxHolder.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: depositStakeTx.Holder.Address.String()},
				Amount:              &types.Amount{Value: depositStakeTx.Holder.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
			}
			transaction.Operations = append(transaction.Operations, tfuelHolder)
		}

	case WithdrawStake:
		withdrawStakeTx := ttypes.WithdrawStakeTx{}
		json.Unmarshal(rawTx, &withdrawStakeTx)

		transaction.Metadata = map[string]interface{}{
			"fee":     withdrawStakeTx.Fee,
			"purpose": withdrawStakeTx.Purpose,
		}

		var i int64
		transaction.Operations = []*types.Operation{}
		if withdrawStakeTx.Source.Coins.ThetaWei != nil {
			thetaSource := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				Type:                WithdrawStakeTxSource.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Source.Address.String()},
				Amount:              &types.Amount{Value: withdrawStakeTx.Source.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
				Metadata:            map[string]interface{}{"sequence": withdrawStakeTx.Source.Sequence, "signature": withdrawStakeTx.Source.Signature},
			}
			transaction.Operations = append(transaction.Operations, thetaSource)
			i++
		}
		if withdrawStakeTx.Source.Coins.TFuelWei != nil {
			tfuelSource := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				Type:                WithdrawStakeTxSource.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Source.Address.String()},
				Amount:              &types.Amount{Value: withdrawStakeTx.Source.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
				Metadata:            map[string]interface{}{"sequence": withdrawStakeTx.Source.Sequence, "signature": withdrawStakeTx.Source.Signature},
			}
			if i > 0 {
				tfuelSource.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
			}
			transaction.Operations = append(transaction.Operations, tfuelSource)
			i++
		}
		if withdrawStakeTx.Holder.Coins.ThetaWei != nil {
			thetaHolder := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
				Type:                WithdrawStakeTxHolder.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Holder.Address.String()},
				Amount:              &types.Amount{Value: withdrawStakeTx.Holder.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			}
			transaction.Operations = append(transaction.Operations, thetaHolder)
			i++
		}
		if withdrawStakeTx.Holder.Coins.TFuelWei != nil {
			tfuelHolder := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
				Type:                WithdrawStakeTxHolder.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Holder.Address.String()},
				Amount:              &types.Amount{Value: withdrawStakeTx.Holder.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
			}
			transaction.Operations = append(transaction.Operations, tfuelHolder)
		}

	case StakeRewardDistribution:
		stakeRewardDistributionTx := ttypes.StakeRewardDistributionTx{}
		json.Unmarshal(rawTx, &stakeRewardDistributionTx)

		transaction.Metadata = map[string]interface{}{
			"fee":               stakeRewardDistributionTx.Fee,
			"split_basis_point": stakeRewardDistributionTx.SplitBasisPoint,
		}

		var i int64
		transaction.Operations = []*types.Operation{}
		if stakeRewardDistributionTx.Holder.Coins.ThetaWei != nil {
			thetaInput := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				Type:                StakeRewardDistributionTxHolder.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Holder.Address.String()},
				Amount:              &types.Amount{Value: stakeRewardDistributionTx.Holder.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
				Metadata:            map[string]interface{}{"sequence": stakeRewardDistributionTx.Holder.Sequence, "signature": stakeRewardDistributionTx.Holder.Signature},
			}
			transaction.Operations = append(transaction.Operations, thetaInput)
			i++
		}
		if stakeRewardDistributionTx.Holder.Coins.TFuelWei != nil {
			tfuelInput := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				Type:                StakeRewardDistributionTxHolder.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Holder.Address.String()},
				Amount:              &types.Amount{Value: stakeRewardDistributionTx.Holder.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
				Metadata:            map[string]interface{}{"sequence": stakeRewardDistributionTx.Holder.Sequence, "signature": stakeRewardDistributionTx.Holder.Signature},
			}
			if i > 0 {
				tfuelInput.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
			}
			transaction.Operations = append(transaction.Operations, tfuelInput)
			i++
		}
		if stakeRewardDistributionTx.Holder.Coins.ThetaWei != nil {
			thetaOutput := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
				Type:                StakeRewardDistributionTxBeneficiary.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Beneficiary.Address.String()},
				Amount:              &types.Amount{Value: stakeRewardDistributionTx.Beneficiary.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			}
			transaction.Operations = append(transaction.Operations, thetaOutput)
			i++
		}
		if stakeRewardDistributionTx.Holder.Coins.TFuelWei != nil {
			tfuelOutput := &types.Operation{
				OperationIdentifier: &types.OperationIdentifier{Index: i},
				RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
				Type:                StakeRewardDistributionTxBeneficiary.String(),
				Status:              status, // same as block status
				Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Beneficiary.Address.String()},
				Amount:              &types.Amount{Value: stakeRewardDistributionTx.Beneficiary.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
			}
			transaction.Operations = append(transaction.Operations, tfuelOutput)
		}
	}

	return transaction
}
