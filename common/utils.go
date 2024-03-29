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
	"github.com/thetatoken/theta/blockchain"
	cmn "github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/crypto/bls"
	ttypes "github.com/thetatoken/theta/ledger/types"
	jrpc "github.com/ybbus/jsonrpc"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "utils"})

const StakeWithdrawPrefix = "stake_withdraw"

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
		if !strings.EqualFold(ni.Blockchain, ChainName) {
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
	SnapshotBlockHeight        cmn.JSONUint64 `json:"snapshot_block_height"`
	SnapshotBlockHash          cmn.Hash       `json:"snapshot_block_hash"`
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

// ------------------------------ BlockIdentifier -----------------------------------

type GetBlockIdentifierByHashArgs struct {
	Hash cmn.Hash `json:"hash"`
}

type GetBlockIdentifierByHeightArgs struct {
	Height cmn.JSONUint64 `json:"height"`
}

type GetBlockIdentifierResultInner struct {
	Height cmn.JSONUint64 `json:"height"`
	Hash   cmn.Hash       `json:"hash"`
}

type GetBlocIdentifierResult struct {
	*GetBlockIdentifierResultInner
}

func GetBlockIdentifierByHeight(client jrpc.RPCClient, height cmn.JSONUint64) (*GetBlocIdentifierResult, *types.Error) {
	rpcRes, rpcErr := client.Call("theta.GetBlockByHeight", GetBlockIdentifierByHeightArgs{
		Height: height,
	})
	if rpcErr != nil {
		return nil, &types.Error{Message: fmt.Sprintf("failed to get block by height: %s", rpcErr.Error())}
	}
	if rpcRes != nil && rpcRes.Error != nil {
		if rpcRes.Error.Code == -32000 {
			return nil, ErrUnableToGetBlk
		} else {
			return nil, &types.Error{Message: fmt.Sprintf("failed to get block by height: %s", rpcRes.Error)}
		}
	}
	return parseBlockIdentifierResult(rpcRes)
}

func GetBlockIdentifierByHash(client jrpc.RPCClient, hash string) (*GetBlocIdentifierResult, *types.Error) {
	rpcRes, rpcErr := client.Call("theta.GetBlock", GetBlockIdentifierByHashArgs{
		Hash: cmn.HexToHash(hash),
	})
	if rpcErr != nil {
		return nil, &types.Error{Message: fmt.Sprintf("failed to get block by hash: %s", rpcErr.Error())}
	}
	if rpcRes != nil && rpcRes.Error != nil {
		if rpcRes.Error.Code == -32000 {
			return nil, ErrUnableToGetBlk
		} else {
			return nil, &types.Error{Message: fmt.Sprintf("failed to get block by hash: %s", rpcRes.Error)}
		}
	}
	return parseBlockIdentifierResult(rpcRes)
}

func parseBlockIdentifierResult(rpcRes *jrpc.RPCResponse) (*GetBlocIdentifierResult, *types.Error) {
	jsonBytes, err := json.MarshalIndent(rpcRes.Result, "", "    ")
	if err != nil {
		return nil, &types.Error{Message: fmt.Sprintf("failed to parse theta RPC response: %v, %s", err, string(jsonBytes))}
	}

	trpcResult := GetBlocIdentifierResult{}
	json.Unmarshal(jsonBytes, &trpcResult)

	if trpcResult.GetBlockIdentifierResultInner == nil {
		return nil, ErrUnableToGetBlk
	}
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
		Amount:              &types.Amount{Value: "0", Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": coinbaseTx.Proposer.Sequence, "signature": sigBytes},
	}
	if status != nil {
		op.Status = status
	}

	ops = []*types.Operation{&op}

	for i, output := range coinbaseTx.Outputs {
		outputOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: int64(i) + 1},
			RelatedOperations:   []*types.OperationIdentifier{{Index: 0}},
			Type:                CoinbaseTxOutput.String(),
			Account:             &types.AccountIdentifier{Address: output.Address.String()},
			Amount:              &types.Amount{Value: output.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
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
		"slash_proof":      slashTx.SlashProof,
	}

	sigBytes, _ := slashTx.Proposer.Signature.MarshalJSON()
	op := types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                SlashTxProposer.String(),
		Status:              status, // same as block status
		Account:             &types.AccountIdentifier{Address: slashTx.Proposer.Address.String()},
		Amount:              &types.Amount{Value: "0", Currency: GetThetaCurrency()},
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
		"fee":  sendTx.Fee,
	}

	var i int64
	var inputAddr string

	for _, input := range sendTx.Inputs {
		sigBytes, _ := input.Signature.MarshalJSON()
		inputAddr = input.Address.String()

		if input.Coins.ThetaWei == nil {
			input.Coins.ThetaWei = big.NewInt(0)
		}
		thetaWei := new(big.Int).Mul(input.Coins.ThetaWei, big.NewInt(-1)).String()
		thetaInputOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                SendTxInput.String(),
			Account:             &types.AccountIdentifier{Address: input.Address.String()},
			Amount:              &types.Amount{Value: thetaWei, Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": input.Sequence, "signature": sigBytes},
		}
		if status != nil {
			thetaInputOp.Status = status
		}
		if i > 0 {
			thetaInputOp.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, thetaInputOp)
		i++

		if input.Coins.TFuelWei == nil {
			input.Coins.TFuelWei = big.NewInt(0)
		}
		tfuelInput := new(big.Int).Sub(input.Coins.TFuelWei, sendTx.Fee.TFuelWei)
		tfuelWei := new(big.Int).Mul(tfuelInput, big.NewInt(-1)).String()
		tfuelInputOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                SendTxInput.String(),
			Account:             &types.AccountIdentifier{Address: input.Address.String()},
			Amount:              &types.Amount{Value: tfuelWei, Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": input.Sequence, "signature": sigBytes},
		}
		if status != nil {
			tfuelInputOp.Status = status
		}
		if i > 0 {
			tfuelInputOp.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, tfuelInputOp)
		i++
	}

	for _, output := range sendTx.Outputs {
		if output.Coins.ThetaWei == nil {
			output.Coins.ThetaWei = big.NewInt(0)
		}
		thetaWei := output.Coins.ThetaWei.String()
		thetaOutputOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{{Index: i - 1}},
			Type:                SendTxOutput.String(),
			Account:             &types.AccountIdentifier{Address: output.Address.String()},
			Amount:              &types.Amount{Value: thetaWei, Currency: GetThetaCurrency()},
		}
		if status != nil {
			thetaOutputOp.Status = status
		}
		if i > 0 {
			thetaOutputOp.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, thetaOutputOp)
		i++

		if output.Coins.TFuelWei == nil {
			output.Coins.TFuelWei = big.NewInt(0)
		}
		tfuelWei := output.Coins.TFuelWei.String()
		tfuelOutputOp := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{{Index: i - 1}},
			Type:                SendTxOutput.String(),
			Account:             &types.AccountIdentifier{Address: output.Address.String()},
			Amount:              &types.Amount{Value: tfuelWei, Currency: GetTFuelCurrency()},
		}
		if status != nil {
			tfuelOutputOp.Status = status
		}
		if i > 0 {
			tfuelOutputOp.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, tfuelOutputOp)
		i++
	}

	fee := new(big.Int).Mul(sendTx.Fee.TFuelWei, big.NewInt(-1)).String()
	feeOp := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: i},
		Type:                TxFee.String(),
		Account:             &types.AccountIdentifier{Address: inputAddr},
		Amount:              &types.Amount{Value: fee, Currency: GetTFuelCurrency()},
	}
	if status != nil {
		feeOp.Status = status
	}
	if i > 0 {
		feeOp.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
	}
	ops = append(ops, feeOp)

	return
}

func ParseReserveFundTx(reserveFundTx ttypes.ReserveFundTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":         txType,
		"collateral":   reserveFundTx.Collateral,
		"resource_ids": reserveFundTx.ResourceIDs,
		"duration":     reserveFundTx.Duration,
	}

	sigBytes, _ := reserveFundTx.Source.Signature.MarshalJSON()
	var i int64

	if reserveFundTx.Source.Coins.ThetaWei != nil && reserveFundTx.Source.Coins.ThetaWei != big.NewInt(0) {
		op := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                ReserveFundTxSource.String(),
			Account:             &types.AccountIdentifier{Address: reserveFundTx.Source.Address.String()},
			Amount:              &types.Amount{Value: new(big.Int).Mul(reserveFundTx.Source.Coins.ThetaWei, big.NewInt(-1)).String(), Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": reserveFundTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			op.Status = status
		}
		ops = append(ops, op)
		i++
	}

	if reserveFundTx.Source.Coins.TFuelWei != nil && reserveFundTx.Source.Coins.TFuelWei != big.NewInt(0) {
		op := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                ReserveFundTxSource.String(),
			Account:             &types.AccountIdentifier{Address: reserveFundTx.Source.Address.String()},
			Amount:              &types.Amount{Value: new(big.Int).Mul(reserveFundTx.Source.Coins.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": reserveFundTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			op.Status = status
		}
		if i > 0 {
			op.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, op)
		i++
	}

	fee := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: i},
		Type:                TxFee.String(),
		Account:             &types.AccountIdentifier{Address: reserveFundTx.Source.Address.String()},
		Amount:              &types.Amount{Value: new(big.Int).Mul(reserveFundTx.Fee.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
		RelatedOperations:   []*types.OperationIdentifier{{Index: 1}},
	}
	if status != nil {
		fee.Status = status
	}
	if i > 0 {
		fee.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
	}
	ops = append(ops, fee)

	return
}

func ParseReleaseFundTx(releaseFundTx ttypes.ReleaseFundTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":             txType,
		"reserve_sequence": releaseFundTx.ReserveSequence,
	}

	sigBytes, _ := releaseFundTx.Source.Signature.MarshalJSON()
	var i int64

	if releaseFundTx.Source.Coins.ThetaWei != nil && releaseFundTx.Source.Coins.ThetaWei != big.NewInt(0) {
		op := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                ReleaseFundTxSource.String(),
			Account:             &types.AccountIdentifier{Address: releaseFundTx.Source.Address.String()},
			Amount:              &types.Amount{Value: releaseFundTx.Source.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": releaseFundTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			op.Status = status
		}
		ops = append(ops, op)
		i++
	}

	if releaseFundTx.Source.Coins.TFuelWei != nil && releaseFundTx.Source.Coins.TFuelWei != big.NewInt(0) {
		op := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                ReleaseFundTxSource.String(),
			Account:             &types.AccountIdentifier{Address: releaseFundTx.Source.Address.String()},
			Amount:              &types.Amount{Value: new(big.Int).Mul(releaseFundTx.Source.Coins.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": releaseFundTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			op.Status = status
		}
		if i > 0 {
			op.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, op)
		i++
	}

	fee := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: i},
		Type:                TxFee.String(),
		Account:             &types.AccountIdentifier{Address: releaseFundTx.Source.Address.String()},
		Amount:              &types.Amount{Value: new(big.Int).Mul(releaseFundTx.Fee.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
		RelatedOperations:   []*types.OperationIdentifier{{Index: 1}},
	}
	if status != nil {
		fee.Status = status
	}
	if i > 0 {
		fee.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
	}
	ops = append(ops, fee)

	return
}

func ParseServicePaymentTx(servicePaymentTx ttypes.ServicePaymentTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":             txType,
		"payment_sequence": servicePaymentTx.PaymentSequence,
		"reserve_sequence": servicePaymentTx.ReserveSequence,
		"resource_id":      servicePaymentTx.ResourceID,
	}

	sigBytes, _ := servicePaymentTx.Source.Signature.MarshalJSON()

	sourceOp := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                ServicePaymentTxSource.String(),
		Account:             &types.AccountIdentifier{Address: servicePaymentTx.Source.Address.String()},
		Amount:              &types.Amount{Value: new(big.Int).Mul(servicePaymentTx.Source.Coins.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": servicePaymentTx.Source.Sequence, "signature": sigBytes},
	}

	targetOp := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 1},
		Type:                ServicePaymentTxTarget.String(),
		Account:             &types.AccountIdentifier{Address: servicePaymentTx.Target.Address.String()},
		Amount:              &types.Amount{Value: servicePaymentTx.Target.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		RelatedOperations:   []*types.OperationIdentifier{{Index: 0}},
	}

	fee := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 2},
		Type:                TxFee.String(),
		Account:             &types.AccountIdentifier{Address: servicePaymentTx.Target.Address.String()},
		Amount:              &types.Amount{Value: new(big.Int).Mul(servicePaymentTx.Fee.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
		RelatedOperations:   []*types.OperationIdentifier{{Index: 1}},
	}

	if status != nil {
		sourceOp.Status = status
		targetOp.Status = status
		fee.Status = status
	}
	ops = []*types.Operation{sourceOp, targetOp, fee}
	return
}

func ParseSplitRuleTx(splitRuleTx ttypes.SplitRuleTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":        txType,
		"resource_id": splitRuleTx.ResourceID,
		"splits":      splitRuleTx.Splits,
		"duration":    splitRuleTx.Duration,
	}

	sigBytes, _ := splitRuleTx.Initiator.Signature.MarshalJSON()
	op := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                SplitRuleTxInitiator.String(),
		Account:             &types.AccountIdentifier{Address: splitRuleTx.Initiator.Address.String()},
		Amount:              &types.Amount{Value: splitRuleTx.Initiator.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		Metadata:            map[string]interface{}{"sequence": splitRuleTx.Initiator.Sequence, "signature": sigBytes},
	}

	fee := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 1},
		Type:                TxFee.String(),
		Account:             &types.AccountIdentifier{Address: splitRuleTx.Initiator.Address.String()},
		Amount:              &types.Amount{Value: new(big.Int).Mul(splitRuleTx.Fee.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
		RelatedOperations:   []*types.OperationIdentifier{{Index: 0}},
	}

	if status != nil {
		op.Status = status
		fee.Status = status
	}
	ops = []*types.Operation{op, fee}
	return
}

func ParseSmartContractTxForConstruction(smartContractTx ttypes.SmartContractTx, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":      txType,
		"gas_limit": smartContractTx.GasLimit,
		"gas_price": smartContractTx.GasPrice,
		"data":      smartContractTx.Data,
	}

	sigBytes, _ := smartContractTx.From.Signature.MarshalJSON()
	var i int64

	if smartContractTx.From.Coins.ThetaWei != nil && len(smartContractTx.From.Coins.ThetaWei.Bits()) != 0 {
		thetaFrom := types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                SmartContractTxFrom.String(),
			Account:             &types.AccountIdentifier{Address: smartContractTx.From.Address.String()},
			Amount:              &types.Amount{Value: new(big.Int).Mul(smartContractTx.From.Coins.ThetaWei, big.NewInt(-1)).String(), Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": smartContractTx.From.Sequence, "signature": sigBytes},
		}
		if i > 0 {
			thetaFrom.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, &thetaFrom)
		i++
	}
	if smartContractTx.From.Coins.TFuelWei != nil && len(smartContractTx.From.Coins.TFuelWei.Bits()) != 0 {
		tfuelFrom := types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                SmartContractTxFrom.String(),
			Account:             &types.AccountIdentifier{Address: smartContractTx.From.Address.String()},
			Amount:              &types.Amount{Value: new(big.Int).Mul(smartContractTx.From.Coins.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": smartContractTx.From.Sequence, "signature": sigBytes},
		}
		if i > 0 {
			tfuelFrom.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, &tfuelFrom)
		i++
	}

	if smartContractTx.To.Coins.ThetaWei != nil && len(smartContractTx.To.Coins.ThetaWei.Bits()) != 0 {
		thetaTo := types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                SmartContractTxTo.String(),
			Account:             &types.AccountIdentifier{Address: smartContractTx.To.Address.String()},
			Amount:              &types.Amount{Value: smartContractTx.To.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
		}
		if i > 0 {
			thetaTo.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, &thetaTo)
		i++
	}
	if smartContractTx.To.Coins.TFuelWei != nil && len(smartContractTx.To.Coins.TFuelWei.Bits()) != 0 {
		tfuelTo := types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                SmartContractTxTo.String(),
			Account:             &types.AccountIdentifier{Address: smartContractTx.To.Address.String()},
			Amount:              &types.Amount{Value: smartContractTx.To.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		}
		if i > 0 {
			tfuelTo.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, &tfuelTo)
		i++
	}
	return
}

func ParseSmartContractTx(smartContractTx ttypes.SmartContractTx, status *string, txType TxType, gasUsed uint64, balanceChanges *blockchain.TxBalanceChangesEntry) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":      txType,
		"gas_limit": smartContractTx.GasLimit,
		"gas_price": smartContractTx.GasPrice,
		"data":      smartContractTx.Data,
	}

	var i int64

	for _, balanceChange := range balanceChanges.BalanceChanges {
		if balanceChange.TokenType > 1 {
			continue
		}

		var currency *types.Currency
		if balanceChange.TokenType == 0 {
			currency = GetThetaCurrency()
		} else {
			currency = GetTFuelCurrency()
		}

		amount := "0"
		var opType string
		if balanceChange.IsNegative {
			amount = new(big.Int).Mul(balanceChange.Delta, big.NewInt(-1)).String()
			opType = SmartContractTxFrom.String()
		} else {
			amount = balanceChange.Delta.String()
			opType = SmartContractTxTo.String()
		}

		op := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                opType,
			Account:             &types.AccountIdentifier{Address: balanceChange.Address.String()},
			Amount:              &types.Amount{Value: amount, Currency: currency},
		}

		if status != nil {
			op.Status = status
		}
		if i > 0 {
			op.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, op)
		i++
	}

	if gasUsed != 0 {
		txFee := new(big.Int).Mul(new(big.Int).Mul(smartContractTx.GasPrice, new(big.Int).SetUint64(gasUsed)), big.NewInt(-1)).String()
		fee := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                TxFee.String(),
			Account:             &types.AccountIdentifier{Address: smartContractTx.From.Address.String()},
			Amount:              &types.Amount{Value: txFee, Currency: GetTFuelCurrency()},
		}
		if status != nil {
			fee.Status = status
		}
		if i > 0 {
			fee.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, fee)
	}

	return
}

func ParseDepositStakeTx(depositStakeTx ttypes.DepositStakeTxV2, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":    txType,
		"purpose": depositStakeTx.Purpose,
	}
	if depositStakeTx.BlsPubkey != nil {
		metadata["bls_pub_key"] = depositStakeTx.BlsPubkey
	}
	if depositStakeTx.BlsPop != nil {
		metadata["bls_pop"] = depositStakeTx.BlsPop
	}
	if depositStakeTx.HolderSig != nil {
		metadata["holder_sig"] = depositStakeTx.HolderSig
	}

	sigBytes, _ := depositStakeTx.Source.Signature.MarshalJSON()
	var i int64

	if depositStakeTx.Source.Coins.ThetaWei != nil && depositStakeTx.Source.Coins.ThetaWei != big.NewInt(0) {
		thetaSource := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                DepositStakeTxSource.String(),
			Account:             &types.AccountIdentifier{Address: depositStakeTx.Source.Address.String()},
			Amount:              &types.Amount{Value: new(big.Int).Mul(depositStakeTx.Source.Coins.ThetaWei, big.NewInt(-1)).String(), Currency: GetThetaCurrency()},
			Metadata:            map[string]interface{}{"sequence": depositStakeTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			thetaSource.Status = status
		}
		if i > 0 {
			thetaSource.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, thetaSource)
		i++
	}

	if depositStakeTx.Source.Coins.TFuelWei != nil && depositStakeTx.Source.Coins.TFuelWei != big.NewInt(0) {
		tfuelSource := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			Type:                DepositStakeTxSource.String(),
			Account:             &types.AccountIdentifier{Address: depositStakeTx.Source.Address.String()},
			Amount:              &types.Amount{Value: new(big.Int).Mul(depositStakeTx.Source.Coins.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
			Metadata:            map[string]interface{}{"sequence": depositStakeTx.Source.Sequence, "signature": sigBytes},
		}
		if status != nil {
			tfuelSource.Status = status
		}
		if i > 0 {
			tfuelSource.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, tfuelSource)
		i++
	}

	fee := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: i},
		Type:                TxFee.String(),
		Account:             &types.AccountIdentifier{Address: depositStakeTx.Source.Address.String()},
		Amount:              &types.Amount{Value: new(big.Int).Mul(depositStakeTx.Fee.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
	}
	if status != nil {
		fee.Status = status
	}
	if i > 0 {
		fee.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
	}
	ops = append(ops, fee)

	return
}

func ParseWithdrawStakeTx(withdrawStakeTx ttypes.WithdrawStakeTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":    txType,
		"purpose": withdrawStakeTx.Purpose,
	}

	fee := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: 0},
		Type:                TxFee.String(),
		Account:             &types.AccountIdentifier{Address: withdrawStakeTx.Source.Address.String()},
		Amount:              &types.Amount{Value: new(big.Int).Mul(withdrawStakeTx.Fee.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
	}
	if status != nil {
		fee.Status = status
	}
	ops = append(ops, fee)
	return
}

func ParseReturnStakeTx(withdrawStakeTx ttypes.WithdrawStakeTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":    txType,
		"purpose": withdrawStakeTx.Purpose,
	}

	sigBytes, _ := withdrawStakeTx.Source.Signature.MarshalJSON()
	var i int64

	if withdrawStakeTx.Source.Coins.ThetaWei != nil && withdrawStakeTx.Source.Coins.ThetaWei != big.NewInt(0) {
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
		if i > 0 {
			thetaSource.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, thetaSource)
		i++
	}

	if withdrawStakeTx.Source.Coins.TFuelWei != nil && withdrawStakeTx.Source.Coins.TFuelWei != big.NewInt(0) {
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
			tfuelSource.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, tfuelSource)
	}

	return
}

func ParseStakeRewardDistributionTx(stakeRewardDistributionTx ttypes.StakeRewardDistributionTx, status *string, txType TxType) (metadata map[string]interface{}, ops []*types.Operation) {
	metadata = map[string]interface{}{
		"type":              txType,
		"split_basis_point": stakeRewardDistributionTx.SplitBasisPoint,
	}

	var i int64

	if stakeRewardDistributionTx.Beneficiary.Coins.ThetaWei != nil && stakeRewardDistributionTx.Beneficiary.Coins.ThetaWei != big.NewInt(0) {
		thetaOutput := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{{Index: i - 1}},
			Type:                StakeRewardDistributionTxBeneficiary.String(),
			Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Beneficiary.Address.String()},
			Amount:              &types.Amount{Value: stakeRewardDistributionTx.Beneficiary.Coins.ThetaWei.String(), Currency: GetThetaCurrency()},
		}
		if status != nil {
			thetaOutput.Status = status
		}
		if i > 0 {
			thetaOutput.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, thetaOutput)
		i++
	}

	if stakeRewardDistributionTx.Beneficiary.Coins.TFuelWei != nil && stakeRewardDistributionTx.Beneficiary.Coins.TFuelWei != big.NewInt(0) {
		tfuelOutput := &types.Operation{
			OperationIdentifier: &types.OperationIdentifier{Index: i},
			RelatedOperations:   []*types.OperationIdentifier{{Index: i - 1}},
			Type:                StakeRewardDistributionTxBeneficiary.String(),
			Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Beneficiary.Address.String()},
			Amount:              &types.Amount{Value: stakeRewardDistributionTx.Beneficiary.Coins.TFuelWei.String(), Currency: GetTFuelCurrency()},
		}
		if status != nil {
			tfuelOutput.Status = status
		}
		if i > 0 {
			tfuelOutput.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
		}
		ops = append(ops, tfuelOutput)
		i++
	}

	fee := &types.Operation{
		OperationIdentifier: &types.OperationIdentifier{Index: i},
		Type:                TxFee.String(),
		Account:             &types.AccountIdentifier{Address: stakeRewardDistributionTx.Holder.Address.String()},
		Amount:              &types.Amount{Value: new(big.Int).Mul(stakeRewardDistributionTx.Fee.TFuelWei, big.NewInt(-1)).String(), Currency: GetTFuelCurrency()},
	}
	if status != nil {
		fee.Status = status
	}
	if i > 0 {
		fee.RelatedOperations = []*types.OperationIdentifier{{Index: i - 1}}
	}
	ops = append(ops, fee)

	return
}

func ParseTx(txType TxType, rawTx json.RawMessage, txHash cmn.Hash, status *string, gasUsed uint64, balanceChanges *blockchain.TxBalanceChangesEntry, db *LDBDatabase, stakeService *StakeService, blockHeight cmn.JSONUint64) types.Transaction {
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
		transaction.Metadata, transaction.Operations = ParseSmartContractTx(smartContractTx, status, txType, gasUsed, balanceChanges)
	case DepositStakeTx, DepositStakeV2Tx:
		depositStakeTx := ttypes.DepositStakeTxV2{}
		json.Unmarshal(rawTx, &depositStakeTx)
		transaction.Metadata, transaction.Operations = ParseDepositStakeTx(depositStakeTx, status, txType)
	case WithdrawStakeTx:
		withdrawStakeTx := ttypes.WithdrawStakeTx{}
		json.Unmarshal(rawTx, &withdrawStakeTx)
		transaction.Metadata, transaction.Operations = ParseWithdrawStakeTx(withdrawStakeTx, status, txType)

		if db != nil && stakeService != nil {
			// Stakes are returned 28800 blocks later, so store tx in db for later processing
			// Query gcp/vcp/eenp to get real withdraw amount
			stake, err := stakeService.GetStakeForTx(withdrawStakeTx, blockHeight)
			if err == nil {
				if withdrawStakeTx.Purpose == core.StakeForValidator || withdrawStakeTx.Purpose == core.StakeForGuardian {
					withdrawStakeTx.Source.Coins = ttypes.Coins{ThetaWei: stake.Amount}
				} else {
					withdrawStakeTx.Source.Coins = ttypes.Coins{TFuelWei: stake.Amount}
				}

				returnStakeTx := &ReturnStakeTx{Hash: txHash.Hex(), Tx: withdrawStakeTx}

				returnStakeTxs := ReturnStakeTxs{}
				kvstore := NewKVStore(db)
				heightBytes := new(big.Int).SetUint64(uint64(blockHeight) + core.ReturnLockingPeriod).Bytes()
				if kvstore.Get(heightBytes, &returnStakeTxs) == nil {
					exists := false
					for _, stake := range returnStakeTxs.ReturnStakes {
						if returnStakeTx.Hash == stake.Hash {
							exists = true
						}
					}
					if !exists {
						returnStakeTxs.ReturnStakes = append(returnStakeTxs.ReturnStakes, returnStakeTx)
					}
				} else {
					returnStakeTxs.ReturnStakes = []*ReturnStakeTx{returnStakeTx}
				}
				err = kvstore.Put(heightBytes, returnStakeTxs)
				if err != nil {
					str := fmt.Sprintf("Failed to put stakes for %s at %d, err: %v", txHash.Hex(), heightBytes, err)
					panic(str)
				}
			} else {
				errorString := fmt.Sprintf("Failed to get stakes for %s at %d", txHash.Hex(), blockHeight)
				panic(errorString)
			}
		} else {
			panic("return_stakes db or service is nil")
		}

		transaction.TransactionIdentifier.Hash = crypto.Keccak256Hash([]byte(fmt.Sprintf("%s_%s", StakeWithdrawPrefix, txHash.Hex()))).Hex()
	case StakeRewardDistributionTx:
		stakeRewardDistributionTx := ttypes.StakeRewardDistributionTx{}
		json.Unmarshal(rawTx, &stakeRewardDistributionTx)
		transaction.Metadata, transaction.Operations = ParseStakeRewardDistributionTx(stakeRewardDistributionTx, status, txType)
	}

	return transaction
}

func AssembleTx(ops []*types.Operation, meta map[string]interface{}) (tx ttypes.Tx, err error) {
	var ok bool
	var typ, seq interface{}
	if typ, ok = meta["type"]; !ok {
		return nil, fmt.Errorf("missing tx type")
	}
	if seq, ok = meta["sequence"]; !ok {
		return nil, fmt.Errorf("missing tx sequence")
	}

	txType := TxType(typ.(float64))
	sequence := uint64(seq.(float64))
	fee := ttypes.Coins{TFuelWei: big.NewInt(int64(meta["fee"].(float64)))}

	switch txType {
	case CoinbaseTx:
		inputAmount := new(big.Int)
		inputAmount.SetString(ops[0].Amount.Value, 10)

		outputs := []ttypes.TxOutput{}
		for i := 1; i < len(ops); i++ {
			tFuelWei := new(big.Int)
			tFuelWei.SetString(ops[i].Amount.Value, 10)

			output := ttypes.TxOutput{
				Address: cmn.HexToAddress(ops[i].Account.Address),
				Coins:   ttypes.Coins{TFuelWei: tFuelWei},
			}
			outputs = append(outputs, output)
		}

		tx = &ttypes.CoinbaseTx{
			Proposer: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{TFuelWei: inputAmount},
				Sequence: sequence,
			},
			Outputs: outputs,
		}

	case SlashTx:
		inputAmount := new(big.Int)
		inputAmount.SetString(ops[0].Amount.Value, 10)

		tx = &ttypes.SlashTx{
			Proposer: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{TFuelWei: inputAmount},
				Sequence: sequence,
			},
		}
	case SendTx:
		var inputs []ttypes.TxInput
		var outputs []ttypes.TxOutput

		inputMap := make(map[string][]*types.Operation)
		outputMap := make(map[string][]*types.Operation)
		for _, op := range ops {
			if op.Type == SendTxInput.String() {
				inputMap[op.Account.Address] = append(inputMap[op.Account.Address], op)
			} else if op.Type == SendTxOutput.String() {
				outputMap[op.Account.Address] = append(outputMap[op.Account.Address], op)
			}
		}

		for addr, ops := range inputMap {
			input := ttypes.TxInput{
				Address:  cmn.HexToAddress(addr),
				Sequence: sequence,
			}
			for _, op := range ops {
				coin := new(big.Int)
				coin.SetString(op.Amount.Value, 10)
				if strings.EqualFold(op.Amount.Currency.Symbol, GetThetaCurrency().Symbol) {
					input.Coins.ThetaWei = coin
				} else if strings.EqualFold(op.Amount.Currency.Symbol, GetTFuelCurrency().Symbol) {
					input.Coins.TFuelWei = coin
				}
			}
			inputs = append(inputs, input)
		}
		for addr, ops := range outputMap {
			output := ttypes.TxOutput{
				Address: cmn.HexToAddress(addr),
			}
			for _, op := range ops {
				coin := new(big.Int)
				coin.SetString(op.Amount.Value, 10)
				if strings.EqualFold(op.Amount.Currency.Symbol, GetThetaCurrency().Symbol) {
					output.Coins.ThetaWei = coin
				} else if strings.EqualFold(op.Amount.Currency.Symbol, GetTFuelCurrency().Symbol) {
					output.Coins.TFuelWei = coin
				}
			}
			outputs = append(outputs, output)
		}

		tx = &ttypes.SendTx{
			Fee:     fee,
			Inputs:  inputs,
			Outputs: outputs,
		}

	case ReserveFundTx:
		inputAmount := new(big.Int)
		inputAmount.SetString(ops[0].Amount.Value, 10)

		tx = &ttypes.ReserveFundTx{
			Fee: fee,
			Source: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{TFuelWei: inputAmount},
				Sequence: sequence,
			},
			Collateral:  meta["collateral"].(ttypes.Coins),
			ResourceIDs: meta["resource_ids"].([]string),
			Duration:    uint64(meta["duration"].(float64)),
		}

	case ReleaseFundTx:
		inputAmount := new(big.Int)
		inputAmount.SetString(ops[0].Amount.Value, 10)

		tx = &ttypes.ReleaseFundTx{
			Fee: fee,
			Source: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{TFuelWei: inputAmount},
				Sequence: sequence,
			},
			ReserveSequence: uint64(meta["reserve_sequence"].(float64)),
		}

	case ServicePaymentTx:
		sourceAmount := new(big.Int)
		sourceAmount.SetString(ops[0].Amount.Value, 10)
		targetAmount := new(big.Int)
		targetAmount.SetString(ops[1].Amount.Value, 10)

		tx = &ttypes.ServicePaymentTx{
			Fee: fee,
			Source: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{TFuelWei: sourceAmount},
				Sequence: sequence,
			},
			Target: ttypes.TxInput{
				Address: cmn.HexToAddress(ops[1].Account.Address),
				Coins:   ttypes.Coins{TFuelWei: targetAmount},
			},
			PaymentSequence: uint64(meta["payment_sequence"].(float64)),
			ReserveSequence: uint64(meta["reserve_sequence"].(float64)),
			ResourceID:      meta["resource_id"].(string),
		}

	case SplitRuleTx:
		sourceAmount := new(big.Int)
		sourceAmount.SetString(ops[0].Amount.Value, 10)

		tx = &ttypes.SplitRuleTx{
			Fee: fee,
			Initiator: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{TFuelWei: sourceAmount},
				Sequence: sequence,
			},
			ResourceID: meta["resource_id"].(string),
			Splits:     meta["splits"].([]ttypes.Split),
			Duration:   uint64(meta["duration"].(float64)),
		}

	case SmartContractTx:
		fromTFuelWei := new(big.Int)
		fromTFuelWei.SetString(ops[0].Amount.Value, 10)
		toTFuelWei := new(big.Int)
		toTFuelWei.SetString(ops[1].Amount.Value, 10)

		tx = &ttypes.SmartContractTx{
			From: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{ThetaWei: big.NewInt(0), TFuelWei: fromTFuelWei},
				Sequence: sequence,
			},
			To: ttypes.TxOutput{
				Address: cmn.HexToAddress(ops[1].Account.Address),
				Coins:   ttypes.Coins{ThetaWei: big.NewInt(0), TFuelWei: toTFuelWei},
			},
			GasLimit: uint64(meta["gas_limit"].(float64)),
			GasPrice: big.NewInt(int64(meta["gas_price"].(float64))), //new(big.Int).Set(meta["gas_price"].(*big.Int)),
			Data:     meta["data"].([]byte),
		}

	case DepositStakeTx, DepositStakeV2Tx:
		sourceThetaWei := new(big.Int)
		sourceThetaWei.SetString(ops[0].Amount.Value, 10)
		sourceTFuelWei := new(big.Int)
		sourceTFuelWei.SetString(ops[1].Amount.Value, 10)
		holderThetaWei := new(big.Int)
		holderThetaWei.SetString(ops[2].Amount.Value, 10)
		holderTFuelWei := new(big.Int)
		holderTFuelWei.SetString(ops[3].Amount.Value, 10)

		depositStakeTx := &ttypes.DepositStakeTxV2{
			Fee:     fee,
			Purpose: uint8(meta["purpose"].(float64)),
			Source: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{ThetaWei: sourceThetaWei, TFuelWei: sourceTFuelWei},
				Sequence: sequence,
			},
			Holder: ttypes.TxOutput{
				Address: cmn.HexToAddress(ops[1].Account.Address),
				Coins:   ttypes.Coins{ThetaWei: holderThetaWei, TFuelWei: holderTFuelWei},
			},
		}

		if blsPubkey, ok := meta["bls_pub_key"]; ok {
			depositStakeTx.BlsPubkey = blsPubkey.(*bls.PublicKey)
		}
		if blsPop, ok := meta["bls_pop"]; ok {
			depositStakeTx.BlsPop = blsPop.(*bls.Signature)
		}
		if holderSig, ok := meta["holder_sig"]; ok {
			depositStakeTx.HolderSig = holderSig.(*crypto.Signature)
		}
		tx = depositStakeTx

	case WithdrawStakeTx:
		sourceThetaWei := new(big.Int)
		sourceThetaWei.SetString(ops[0].Amount.Value, 10)
		sourceTFuelWei := new(big.Int)
		sourceTFuelWei.SetString(ops[1].Amount.Value, 10)
		holderThetaWei := new(big.Int)
		holderThetaWei.SetString(ops[2].Amount.Value, 10)
		holderTFuelWei := new(big.Int)
		holderTFuelWei.SetString(ops[3].Amount.Value, 10)

		tx = &ttypes.WithdrawStakeTx{
			Fee:     fee,
			Purpose: uint8(meta["purpose"].(float64)),
			Source: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{ThetaWei: sourceThetaWei, TFuelWei: sourceTFuelWei},
				Sequence: sequence,
			},
			Holder: ttypes.TxOutput{
				Address: cmn.HexToAddress(ops[1].Account.Address),
				Coins:   ttypes.Coins{ThetaWei: holderThetaWei, TFuelWei: holderTFuelWei},
			},
		}

	case StakeRewardDistributionTx:
		holderThetaWei := new(big.Int)
		holderThetaWei.SetString(ops[0].Amount.Value, 10)
		holderTFuelWei := new(big.Int)
		holderTFuelWei.SetString(ops[1].Amount.Value, 10)
		beneficiaryThetaWei := new(big.Int)
		beneficiaryThetaWei.SetString(ops[2].Amount.Value, 10)
		beneficiaryTFuelWei := new(big.Int)
		beneficiaryTFuelWei.SetString(ops[3].Amount.Value, 10)

		tx = &ttypes.StakeRewardDistributionTx{
			Fee:             fee,
			SplitBasisPoint: uint(meta["split_basis_point"].(float64)),
			Holder: ttypes.TxInput{
				Address:  cmn.HexToAddress(ops[0].Account.Address),
				Coins:    ttypes.Coins{ThetaWei: holderThetaWei, TFuelWei: holderTFuelWei},
				Sequence: sequence,
			},
			Beneficiary: ttypes.TxOutput{
				Address: cmn.HexToAddress(ops[1].Account.Address),
				Coins:   ttypes.Coins{ThetaWei: beneficiaryThetaWei, TFuelWei: beneficiaryTFuelWei},
			},
		}

	default:
		return nil, fmt.Errorf("unsupported tx type")
	}
	return
}
