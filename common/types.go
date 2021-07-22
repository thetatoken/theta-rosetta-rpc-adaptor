package common

import (
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	ttypes "github.com/thetatoken/theta/ledger/types"
)

const (
	Theta = "theta"

	// Decimals
	CoinDecimals = 18
)

func GetThetaCurrency() *types.Currency {
	return &types.Currency{
		Symbol:   ttypes.DenomThetaWei,
		Decimals: CoinDecimals,
	}
}

func GetTFuelCurrency() *types.Currency {
	return &types.Currency{
		Symbol:   ttypes.DenomTFuelWei,
		Decimals: CoinDecimals,
	}
}

// ------------------------------ Tx Type -----------------------------------

type TxType byte
type TxStatus string

type Tx struct {
	ttypes.Tx `json:"raw"`
	Type      TxType                     `json:"type"`
	Hash      common.Hash                `json:"hash"`
	Receipt   *blockchain.TxReceiptEntry `json:"receipt"`
}

const (
	CoinbaseTx TxType = iota
	SlashTx
	SendTx
	ReserveFundTx
	ReleaseFundTx
	ServicePaymentTx
	SplitRuleTx
	SmartContractTx
	DepositStakeTx
	WithdrawStakeTx
	DepositStakeV2Tx
	StakeRewardDistributionTx
)

func (t TxType) String() string {
	return [...]string{
		"CoinbaseTx",
		"SlashTx",
		"SendTx",
		"ReserveFundTx",
		"ServicePaymentTx",
		"SplitRuleTx",
		"SmartContractTx",
		"DepositStakeTx",
		"WithdrawStakeTx",
		"DepositStakeV2Tx",
		"StakeRewardDistributionTx",
	}[t]
}

func TxTypes() []string {
	return []string{
		"CoinbaseTx",
		"SlashTx",
		"SendTx",
		"ReserveFundTx",
		"ServicePaymentTx",
		"SplitRuleTx",
		"SmartContractTx",
		"DepositStakeTx",
		"WithdrawStakeTx",
		"DepositStakeV2Tx",
		"StakeRewardDistributionTx",
	}
}

// ------------------------------ Tx Operation Type -----------------------------------

type TxOpType byte

const (
	CoinbaseTxProposer TxOpType = iota
	CoinbaseTxOutput
	SlashTxProposer
	SendTxFee
	SendTxInput
	SendTxOutput
	ReserveFundTxSource
	ReleaseFundTxSource
	ServicePaymentTxSource
	ServicePaymentTxTarget
	SplitRuleTxInitiator
	SmartContractTxFrom
	SmartContractTxTo
	DepositStakeTxSource
	DepositStakeTxHolder
	WithdrawStakeTxSource
	WithdrawStakeTxHolder
	StakeRewardDistributionTxHolder
	StakeRewardDistributionTxBeneficiary
)

func (t TxOpType) String() string {
	return [...]string{
		"CoinbaseTxProposer",
		"CoinbaseTxOutput",
		"SlashTxProposer",
		"SendTxFee",
		"SendTxInput",
		"SendTxOutput",
		"ReserveFundTxSource",
		"ReleaseFundTxSource",
		"ServicePaymentTxSource",
		"ServicePaymentTxTarget",
		"SplitRuleTxInitiator",
		"SmartContractTxFrom",
		"SmartContractTxTo",
		"DepositStakeTxSource",
		"DepositStakeTxHolder",
		"WithdrawStakeTxSource",
		"WithdrawStakeTxHolder",
		"StakeRewardDistributionTxHolder",
		"StakeRewardDistributionTxBeneficiary",
	}[t]
}

func TxOpTypes() []string {
	return []string{
		"CoinbaseTxProposer",
		"CoinbaseTxOutput",
		"SlashTxProposer",
		"SendTxFee",
		"SendTxInput",
		"SendTxOutput",
		"ReserveFundTxSource",
		"ReleaseFundTxSource",
		"ServicePaymentTxSource",
		"ServicePaymentTxTarget",
		"SplitRuleTxInitiator",
		"SmartContractTxFrom",
		"SmartContractTxTo",
		"DepositStakeTxSource",
		"DepositStakeTxHolder",
		"WithdrawStakeTxSource",
		"WithdrawStakeTxHolder",
		"StakeRewardDistributionTxHolder",
		"StakeRewardDistributionTxBeneficiary",
	}
}

//TODO: merge these two?
func IsSupportedConstructionType(typ string) bool {
	for _, styp := range TxOpTypes() {
		if typ == styp {
			return true
		}
	}
	return false
}

func GetTxOpType(typ string) (TxOpType, error) {
	for i, styp := range TxOpTypes() {
		if typ == styp {
			return TxOpType(i), nil
		}
	}
	return 0, fmt.Errorf("invalid tx op type")
}

// ------------------------------ Block Status -----------------------------------

type BlockStatus byte

const (
	BlockStatusPending BlockStatus = iota
	BlockStatusValid
	BlockStatusInvalid
	BlockStatusCommitted
	BlockStatusDirectlyFinalized
	BlockStatusIndirectlyFinalized
	BlockStatusTrusted
	BlockStatusDisposed
)

func (s BlockStatus) String() string {
	return [...]string{
		"pending",
		"valid",
		"invalid",
		"committed",
		"directly_finalized",
		"indirectly_finalized",
		"trusted",
		"disposed",
	}[s]
}
