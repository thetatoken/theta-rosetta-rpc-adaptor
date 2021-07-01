package common

import (
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	ttypes "github.com/thetatoken/theta/ledger/types"
)

const (
	Theta = "THETA"

	// Theta Symbol
	ThetaWei = "thetawei"
	// TFuel Symbol
	TFuelWei = "tfuelwei"
	// Decimals
	CoinDecimals = 18

	StatusSuccess = "success"
	StatusFail    = "fail"
)

func GetThetaCurrency() *types.Currency {
	return &types.Currency{
		Symbol:   ThetaWei,
		Decimals: CoinDecimals,
	}
}

func GetTFuelCurrency() *types.Currency {
	return &types.Currency{
		Symbol:   TFuelWei,
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
	Coinbase TxType = iota
	Slash
	Send
	ReserveFund
	ReleaseFund
	ServicePayment
	SplitRule
	SmartContract
	DepositStake
	WithdrawStake
	DepositStakeV2
	StakeRewardDistribution
)

func (t TxType) String() string {
	return [...]string{
		"Coinbase",
		"Slash",
		"Send",
		"ReserveFund",
		"ServicePayment",
		"SplitRule",
		"SmartContract",
		"DepositStake",
		"WithdrawStake",
		"DepositStakeV2",
		"StakeRewardDistribution",
	}[t]
}

func TxTypes() []string {
	return []string{
		"Coinbase",
		"Slash",
		"Send",
		"ReserveFund",
		"ServicePayment",
		"SplitRule",
		"SmartContract",
		"DepositStake",
		"WithdrawStake",
		"DepositStakeV2",
		"StakeRewardDistribution",
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
		"finalized",
		"indirectly finalized",
		"trusted",
		"disposed",
	}[s]
}
