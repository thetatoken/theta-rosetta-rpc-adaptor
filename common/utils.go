package common

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/spf13/viper"

	rpcc "github.com/ybbus/jsonrpc"

	"github.com/coinbase/rosetta-sdk-go/types"
	log "github.com/sirupsen/logrus"
	cmn "github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
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

func ParseCoinbaseTx(coinbaseTx ttypes.CoinbaseTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":         txType,
		"block_height": coinbaseTx.BlockHeight,
	}

	sigBytes, _ := coinbaseTx.Proposer.Signature.MarshalJSON()
	op := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                CoinbaseTxProposer.String(),
		Account:             &types.AccountIdentifier{Address: coinbaseTx.Proposer.Address.String()},
		Amount:              &types.Amount{Value: coinbaseTx.Proposer.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": coinbaseTx.Proposer.Sequence, "signature": sigBytes},
	}
	if status != nil {
		op.Status = status
	}

	ops = []*types.Operation{&op}

	for i, output := range coinbaseTx.Outputs {
		outputOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: int64(i) + 1},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: 0}},
			Type:                CoinbaseTxOutput.String(),
			Account:             &types.AccountIdentifier{Address: output.Address.String()},
			Amount:              &types.Amount{Value: output.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
		}
		if status != nil {
			outputOp.Status = status
		}
		ops = append(ops, outputOp)
	}
	return
}

func ParseSlashTx(slashTx ttypes.SlashTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":             txType,
		"slashed_address":  slashTx.SlashedAddress,
		"reserve_sequence": slashTx.ReserveSequence,
		"slash_proof":      slashTx.SlashProof, //TODO: need to convert to hex string?
	}

	sigBytes, _ := slashTx.Proposer.Signature.MarshalJSON()
	op := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                SlashTxProposer.String(),
		Status:              status, // same as block status
		Account:             &types.AccountIdentifier{Address: slashTx.Proposer.Address.String()},
		Amount:              &types.Amount{Value: slashTx.Proposer.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
		Metadata:            map[string]interface{}{"sequence": slashTx.Proposer.Sequence, "signature": sigBytes},
	}
	if status != nil {
		op.Status = status
	}
	ops = []*types.Operation{&op}
	return
}

func ParseSendTx(sendTx ttypes.SendTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type": txType,
		"fee":  sendTx.Fee.TFuelWei.Int64,
	}

	var i int64
	for _, input := range sendTx.Inputs {
		sigBytes, _ := input.Signature.MarshalJSON()

		thetaWei := "0"
		if input.Coins.ThetaWei != nil {
			thetaWei = input.Coins.ThetaWei.String()
		}
		inputOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                SendTxInput.String(),
			Account:             &types.AccountIdentifier{Address: input.Address.String()},
			Amount:              &types.Amount{Value: thetaWei, Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": input.Sequence, "signature": sigBytes},
		}
		if status != nil {
			inputOp.Status = status
		}
		if i > 0 {
			inputOp.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
		}
		ops = append(ops, inputOp)
		i++

		tfuelWei := "0"
		if input.Coins.TFuelWei != nil {
			tfuelWei = input.Coins.TFuelWei.String()
		}
		inputOp = &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                SendTxInput.String(),
			Account:             &types.AccountIdentifier{Address: input.Address.String()},
			Amount:              &types.Amount{Value: tfuelWei, Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": input.Sequence, "signature": sigBytes},
		}
		if status != nil {
			inputOp.Status = status
		}
		if i > 0 {
			inputOp.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
		}
		ops = append(ops, inputOp)
		i++
	}

	for _, output := range sendTx.Outputs {
		thetaWei := "0"
		if output.Coins.ThetaWei != nil {
			thetaWei = output.Coins.ThetaWei.String()
		}

		outputOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
			Type:                SendTxOutput.String(),
			Account:             &types.AccountIdentifier{Address: output.Address.String()},
			Amount:              &types.Amount{Value: thetaWei, Currency: GetThetaCurrency()},
		}
		if status != nil {
			outputOp.Status = status
		}
		ops = append(ops, outputOp)
		i++

		tfuelWei := "0"
		if output.Coins.TFuelWei != nil {
			tfuelWei = output.Coins.TFuelWei.String()
		}

		outputOp = &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
			Type:                SendTxOutput.String(),
			Account:             &types.AccountIdentifier{Address: output.Address.String()},
			Amount:              &types.Amount{Value: tfuelWei, Currency: GetTFuelCurrency()},
		}
		if status != nil {
			outputOp.Status = status
		}
		ops = append(ops, outputOp)
		i++
	}
	return
}

func ParseReserveFundTx(reserveFundTx ttypes.ReserveFundTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":         txType,
		"fee":          reserveFundTx.Fee,
		"collateral":   reserveFundTx.Collateral,
		"resource_ids": reserveFundTx.ResourceIDs,
		"duration":     reserveFundTx.Duration,
	}

	sigBytes, _ := reserveFundTx.Source.Signature.MarshalJSON()
	op := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                ReserveFundTxSource.String(),
		Account:             &types.AccountIdentifier{Address: reserveFundTx.Source.Address.String()},
		Amount:              &types.Amount{Value: reserveFundTx.Source.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": reserveFundTx.Source.Sequence, "signature": sigBytes},
	}
	if status != nil {
		op.Status = status
	}
	ops = []*types.Operation{&op}
	return
}

func ParseReleaseFundTx(releaseFundTx ttypes.ReleaseFundTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":             txType,
		"fee":              releaseFundTx.Fee,
		"reserve_sequence": releaseFundTx.ReserveSequence,
	}

	sigBytes, _ := releaseFundTx.Source.Signature.MarshalJSON()
	op := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                ReleaseFundTxSource.String(),
		Account:             &types.AccountIdentifier{Address: releaseFundTx.Source.Address.String()},
		Amount:              &types.Amount{Value: releaseFundTx.Source.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": releaseFundTx.Source.Sequence, "signature": sigBytes},
	}
	if status != nil {
		op.Status = status
	}
	ops = []*types.Operation{&op}
	return
}

func ParseServicePaymentTx(servicePaymentTx ttypes.ServicePaymentTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":             txType,
		"fee":              servicePaymentTx.Fee,
		"payment_sequence": servicePaymentTx.PaymentSequence,
		"reserve_sequence": servicePaymentTx.ReserveSequence,
		"resource_id":      servicePaymentTx.ResourceID,
	}

	sigBytes, _ := servicePaymentTx.Source.Signature.MarshalJSON()
	sourceOp := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                ServicePaymentTxSource.String(),
		Account:             &types.AccountIdentifier{Address: servicePaymentTx.Source.Address.String()},
		Amount:              &types.Amount{Value: servicePaymentTx.Source.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": servicePaymentTx.Source.Sequence, "signature": sigBytes},
	}
	targetOp := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 1},
		RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: 0}},
		Type:                ServicePaymentTxTarget.String(),
		Account:             &types.AccountIdentifier{Address: servicePaymentTx.Target.Address.String()},
		Amount:              &types.Amount{Value: servicePaymentTx.Target.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		// Metadata:            map[string]interface{}{"sequence": servicePaymentTx.Target.Sequence, "signature": servicePaymentTx.Target.Signature},
	}
	if status != nil {
		sourceOp.Status = status
		targetOp.Status = status
	}
	ops = []*types.Operation{&sourceOp, &targetOp}
	return
}

func ParseSplitRuleTx(splitRuleTx ttypes.SplitRuleTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":        txType,
		"fee":         splitRuleTx.Fee,
		"resource_id": splitRuleTx.ResourceID,
		"splits":      splitRuleTx.Splits, //TODO ?
		"duration":    splitRuleTx.Duration,
	}

	sigBytes, _ := splitRuleTx.Initiator.Signature.MarshalJSON()
	op := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                SplitRuleTxInitiator.String(),
		Account:             &types.AccountIdentifier{Address: splitRuleTx.Initiator.Address.String()},
		Amount:              &types.Amount{Value: splitRuleTx.Initiator.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": splitRuleTx.Initiator.Sequence, "signature": sigBytes},
	}
	if status != nil {
		op.Status = status
	}
	ops = []*types.Operation{&op}
	return
}

func ParseSmartContractTx(smartContractTx ttypes.SmartContractTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":      txType,
		"gas_limit": smartContractTx.GasLimit,
		"gas_price": smartContractTx.GasPrice,
		"data":      smartContractTx.Data,
	}

	sigBytes, _ := smartContractTx.From.Signature.MarshalJSON()
	fromOp := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                SmartContractTxFrom.String(),
		Account:             &types.AccountIdentifier{Address: smartContractTx.From.Address.String()},
		Amount:              &types.Amount{Value: smartContractTx.From.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": smartContractTx.From.Sequence, "signature": sigBytes},
	}
	toOp := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 1},
		RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: 0}},
		Type:                SmartContractTxTo.String(),
		Account:             &types.AccountIdentifier{Address: smartContractTx.To.Address.String()},
		Amount:              &types.Amount{Value: smartContractTx.To.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
	}
	if status != nil {
		fromOp.Status = status
		toOp.Status = status
	}
	ops = []*types.Operation{&fromOp, &toOp}
	return
}

func ParseDepositStakeTx(depositStakeTx ttypes.DepositStakeTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":    txType, //TODO: diff V2?
		"fee":     depositStakeTx.Fee,
		"purpose": depositStakeTx.Purpose,
	}

	sigBytes, _ := depositStakeTx.Source.Signature.MarshalJSON()
	var i int64
	// ops = []*types.Operation{}
	if depositStakeTx.Source.Coins.ThetaWei != nil {
		thetaSource := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                DepositStakeTxSource.String(),
			Account:             &types.AccountIdentifier{Address: depositStakeTx.Source.Address.String()},
			Amount:              &types.Amount{Value: depositStakeTx.Source.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": depositStakeTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			thetaSource.Status = status
		}
		ops = append(ops, thetaSource)
		i++
	}
	if depositStakeTx.Source.Coins.TFuelWei != nil {
		tfuelSource := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                DepositStakeTxSource.String(),
			Account:             &types.AccountIdentifier{Address: depositStakeTx.Source.Address.String()},
			Amount:              &types.Amount{Value: depositStakeTx.Source.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": depositStakeTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			tfuelSource.Status = status
		}
		if i > 0 {
			tfuelSource.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
		}
		ops = append(ops, tfuelSource)
		i++
	}
	if depositStakeTx.Holder.Coins.ThetaWei != nil {
		thetaHolder := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
			Type:                DepositStakeTxHolder.String(),
			Account:             &types.AccountIdentifier{Address: depositStakeTx.Holder.Address.String()},
			Amount:              &types.Amount{Value: depositStakeTx.Holder.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
		}
		if status != nil {
			thetaHolder.Status = status
		}
		ops = append(ops, thetaHolder)
		i++
	}
	if depositStakeTx.Holder.Coins.TFuelWei != nil {
		tfuelHolder := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
			Type:                DepositStakeTxHolder.String(),
			Account:             &types.AccountIdentifier{Address: depositStakeTx.Holder.Address.String()},
			Amount:              &types.Amount{Value: depositStakeTx.Holder.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		}
		if status != nil {
			tfuelHolder.Status = status
		}
		ops = append(ops, tfuelHolder)
	}
	return
}

func ParseWithdrawStakeTx(withdrawStakeTx ttypes.WithdrawStakeTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":    txType,
		"fee":     withdrawStakeTx.Fee,
		"purpose": withdrawStakeTx.Purpose,
	}

	sigBytes, _ := withdrawStakeTx.Source.Signature.MarshalJSON()
	var i int64
	// tx.Operations = []*types.Operation{}
	if withdrawStakeTx.Source.Coins.ThetaWei != nil {
		thetaSource := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                WithdrawStakeTxSource.String(),
			Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Source.Address.String()},
			Amount:              &types.Amount{Value: withdrawStakeTx.Source.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": withdrawStakeTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			thetaSource.Status = status
		}
		ops = append(ops, thetaSource)
		i++
	}
	if withdrawStakeTx.Source.Coins.TFuelWei != nil {
		tfuelSource := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                WithdrawStakeTxSource.String(),
			Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Source.Address.String()},
			Amount:              &types.Amount{Value: withdrawStakeTx.Source.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": withdrawStakeTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			tfuelSource.Status = status
		}
		if i > 0 {
			tfuelSource.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
		}
		ops = append(ops, tfuelSource)
		i++
	}
	if withdrawStakeTx.Holder.Coins.ThetaWei != nil {
		thetaHolder := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
			Type:                WithdrawStakeTxHolder.String(),
			Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Holder.Address.String()},
			Amount:              &types.Amount{Value: withdrawStakeTx.Holder.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
		}
		if status != nil {
			thetaHolder.Status = status
		}
		ops = append(ops, thetaHolder)
		i++
	}
	if withdrawStakeTx.Holder.Coins.TFuelWei != nil {
		tfuelHolder := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
			Type:                WithdrawStakeTxHolder.String(),
			Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Holder.Address.String()},
			Amount:              &types.Amount{Value: withdrawStakeTx.Holder.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		}
		if status != nil {
			tfuelHolder.Status = status
		}
		ops = append(ops, tfuelHolder)
	}
	return
}

func ParseStakeRewardDistributionTx(stakeRewardDistributionTx ttypes.StakeRewardDistributionTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":              txType,
		"fee":               stakeRewardDistributionTx.Fee,
		"split_basis_point": stakeRewardDistributionTx.SplitBasisPoint,
	}

	sigBytes, _ := stakeRewardDistributionTx.Holder.Signature.MarshalJSON()
	var i int64
	ops = []*types.Operation{}
	if stakeRewardDistributionTx.Holder.Coins.ThetaWei != nil {
		thetaInput := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                StakeRewardDistributionTxHolder.String(),
			Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Holder.Address.String()},
			Amount:              &types.Amount{Value: stakeRewardDistributionTx.Holder.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": stakeRewardDistributionTx.Holder.Sequence, "signature": sigBytes},
		}
		if status != nil {
			thetaInput.Status = status
		}
		ops = append(ops, thetaInput)
		i++
	}
	if stakeRewardDistributionTx.Holder.Coins.TFuelWei != nil {
		tfuelInput := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                StakeRewardDistributionTxHolder.String(),
			Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Holder.Address.String()},
			Amount:              &types.Amount{Value: stakeRewardDistributionTx.Holder.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": stakeRewardDistributionTx.Holder.Sequence, "signature": sigBytes},
		}
		if status != nil {
			tfuelInput.Status = status
		}
		if i > 0 {
			tfuelInput.RelatedOperations = []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}}
		}
		ops = append(ops, tfuelInput)
		i++
	}
	if stakeRewardDistributionTx.Holder.Coins.ThetaWei != nil {
		thetaOutput := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
			Type:                StakeRewardDistributionTxBeneficiary.String(),
			Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Beneficiary.Address.String()},
			Amount:              &types.Amount{Value: stakeRewardDistributionTx.Beneficiary.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
		}
		if status != nil {
			thetaOutput.Status = status
		}
		ops = append(ops, thetaOutput)
		i++
	}
	if stakeRewardDistributionTx.Holder.Coins.TFuelWei != nil {
		tfuelOutput := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{&types.OperationIdentifier{Index: i - 1}},
			Type:                StakeRewardDistributionTxBeneficiary.String(),
			Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Beneficiary.Address.String()},
			Amount:              &types.Amount{Value: stakeRewardDistributionTx.Beneficiary.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		}
		if status != nil {
			tfuelOutput.Status = status
		}
		ops = append(ops, tfuelOutput)
	}
	return
}

func ParseTx(txType TxType, rawTx json.RawMessage, txHash cmn.Hash, status *string) types.Transaction {
	transaction := types.Transaction{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: txHash.String()},
	}

	switch txType {
	case CoinbaseTx:
		coinbaseTx := ttypes.CoinbaseTx{}
		json.Unmarshal(rawTx, &coinbaseTx)
		transaction.Metadata, transaction.Operations = ParseCoinbaseTx(coinbaseTx, status, txType)
	case SlashTx:
		slashTx := ttypes.SlashTx{}
		json.Unmarshal(rawTx, &slashTx)
		transaction.Metadata, transaction.Operations = ParseSlashTx(slashTx, status, txType)
	case SendTx:
		sendTx := ttypes.SendTx{}
		json.Unmarshal(rawTx, &sendTx)
		transaction.Metadata, transaction.Operations = ParseSendTx(sendTx, status, txType)
	case ReserveFundTx:
		reserveFundTx := ttypes.ReserveFundTx{}
		json.Unmarshal(rawTx, &reserveFundTx)
		transaction.Metadata, transaction.Operations = ParseReserveFundTx(reserveFundTx, status, txType)
	case ReleaseFundTx:
		releaseFundTx := ttypes.ReleaseFundTx{}
		json.Unmarshal(rawTx, &releaseFundTx)
		transaction.Metadata, transaction.Operations = ParseReleaseFundTx(releaseFundTx, status, txType)
	case ServicePaymentTx:
		servicePaymentTx := ttypes.ServicePaymentTx{}
		json.Unmarshal(rawTx, &servicePaymentTx)
		transaction.Metadata, transaction.Operations = ParseServicePaymentTx(servicePaymentTx, status, txType)
	case SplitRuleTx:
		splitRuleTx := ttypes.SplitRuleTx{}
		json.Unmarshal(rawTx, &splitRuleTx)
		transaction.Metadata, transaction.Operations = ParseSplitRuleTx(splitRuleTx, status, txType)
	case SmartContractTx:
		smartContractTx := ttypes.SmartContractTx{}
		json.Unmarshal(rawTx, &smartContractTx)
		transaction.Metadata, transaction.Operations = ParseSmartContractTx(smartContractTx, status, txType)
	case DepositStakeTx, DepositStakeV2Tx:
		depositStakeTx := ttypes.DepositStakeTx{}
		json.Unmarshal(rawTx, &depositStakeTx)
		transaction.Metadata, transaction.Operations = ParseDepositStakeTx(depositStakeTx, status, txType)
	case WithdrawStakeTx:
		withdrawStakeTx := ttypes.WithdrawStakeTx{}
		json.Unmarshal(rawTx, &withdrawStakeTx)
		transaction.Metadata, transaction.Operations = ParseWithdrawStakeTx(withdrawStakeTx, status, txType)
	case StakeRewardDistributionTx:
		stakeRewardDistributionTx := ttypes.StakeRewardDistributionTx{}
		json.Unmarshal(rawTx, &stakeRewardDistributionTx)
		transaction.Metadata, transaction.Operations = ParseStakeRewardDistributionTx(stakeRewardDistributionTx, status, txType)
	}

	return transaction
}

func AssembleTx(ops []*types.Operation, meta map[string]interface{}) (tx ttypes.Tx, err error) {
	txType := meta["type"]

	switch txType {
	case CoinbaseTx:
		inputAmount := new(big.Int)
		inputAmount.SetString(ops[0].Amount.Value, 10)
		sig := &crypto.Signature{}
		sig.UnmarshalJSON(ops[0].Metadata["signature"].([]byte))

		tx = &ttypes.CoinbaseTx{
			Proposer: ttypes.TxInput{
				Address:   cmn.HexToAddress(ops[0].Account.Address),
				Coins:     ttypes.Coins{TFuelWei: inputAmount},
				Sequence:  ops[0].Metadata["sequence"].(uint64),
				Signature: sig,
			},
			//TODO...
		}
	case SlashTx:

	case SendTx:
		var inputs []ttypes.TxInput
		var outputs []ttypes.TxOutput
		for i := 0; i < len(ops); i += 2 {
			thetaWei := new(big.Int)
			thetaWei.SetString(ops[i].Amount.Value, 10)
			tfuelWei := new(big.Int)
			tfuelWei.SetString(ops[i+1].Amount.Value, 10)

			if ops[i].Type == SendTxInput.String() {
				sig := &crypto.Signature{}
				sig.UnmarshalJSON(ops[i].Metadata["signature"].([]byte))

				input := ttypes.TxInput{
					Address:   cmn.HexToAddress(ops[i].Account.Address),
					Coins:     ttypes.Coins{ThetaWei: thetaWei, TFuelWei: tfuelWei},
					Sequence:  ops[i].Metadata["sequence"].(uint64),
					Signature: sig,
				}
				inputs = append(inputs, input)
			} else if ops[i].Type == SendTxOutput.String() {
				output := ttypes.TxOutput{
					Address: cmn.HexToAddress(ops[i].Account.Address),
					Coins:   ttypes.Coins{ThetaWei: thetaWei, TFuelWei: tfuelWei},
				}
				outputs = append(outputs, output)
			}
		}
		tx = &ttypes.SendTx{
			Fee:     ttypes.Coins{TFuelWei: big.NewInt(meta["fee"].(int64))},
			Inputs:  inputs,
			Outputs: outputs,
		}

	case ReserveFundTx:

	case ReleaseFundTx:

	case ServicePaymentTx:

	case SplitRuleTx:

	case SmartContractTx:
		sig := &crypto.Signature{}
		sig.UnmarshalJSON(ops[0].Metadata["signature"].([]byte))
		fromTFuelWei := new(big.Int)
		fromTFuelWei.SetString(ops[0].Amount.Value, 10)
		toTFuelWei := new(big.Int)
		toTFuelWei.SetString(ops[1].Amount.Value, 10)

		gasPrice := new(big.Int)
		gasPrice.SetInt64(meta["gas_price"].(int64))

		tx = &ttypes.SmartContractTx{
			From: ttypes.TxInput{
				Address:   cmn.HexToAddress(ops[0].Account.Address),
				Coins:     ttypes.Coins{ThetaWei: big.NewInt(0), TFuelWei: fromTFuelWei},
				Sequence:  ops[0].Metadata["sequence"].(uint64),
				Signature: sig,
			},
			To: ttypes.TxOutput{
				Address: cmn.HexToAddress(ops[1].Account.Address),
				Coins:   ttypes.Coins{ThetaWei: big.NewInt(0), TFuelWei: toTFuelWei},
			},
			GasLimit: meta["gas_limit"].(uint64),
			GasPrice: gasPrice,
			Data:     meta["data"].([]byte),
		}

	case DepositStakeTx, DepositStakeV2Tx:

	case WithdrawStakeTx:

	case StakeRewardDistributionTx:

	}
	return tx, nil
}
