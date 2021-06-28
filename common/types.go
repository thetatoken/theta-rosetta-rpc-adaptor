package common

import (
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	ttypes "github.com/thetatoken/theta/ledger/types"
)

const (
	// Theta Symbol
	Theta = "THETA"
	// TFuel Symbol
	TFuel = "TFUEL"
	// Decimals
	CoinDecimals = 18

	StatusSuccess = "success"
	StatusFail    = "fail"
)

func GetThetaCurrency() *types.Currency {
	return &types.Currency{
		Symbol:   Theta,
		Decimals: CoinDecimals,
	}
}

func GetTFuelCurrency() *types.Currency {
	return &types.Currency{
		Symbol:   TFuel,
		Decimals: CoinDecimals,
	}
}

type Tx struct {
	ttypes.Tx `json:"raw"`
	Type      TxType                     `json:"type"`
	Hash      common.Hash                `json:"hash"`
	Receipt   *blockchain.TxReceiptEntry `json:"receipt"`
}

// ------------------------------ Tx Type -----------------------------------

type TxType byte

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
	DepositStakeTxV2
	StakeRewardDistributionTx
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
		"DepositStakeTxV2",
		"StakeRewardDistributionTx",
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
		"DepositStakeTxV2",
		"StakeRewardDistributionTx",
	}
}

// ------------------------------ Tx Operation Type -----------------------------------

type TxOpType byte

const (
	CoinbaseProposer TxOpType = iota
	CoinbaseOutput
	// Slash
	// Send
	// ReserveFund
	// ReleaseFund
	// ServicePayment
	// SplitRule
	// SmartContract
	// DepositStake
	// WithdrawStake
	// DepositStakeTxV2
	// StakeRewardDistributionTx
)

func (t TxOpType) String() string {
	return [...]string{
		"Coinbase Proposer",
		"Coinbase Output",
		"Slash",
		"Send",
		"ReserveFund",
		"ServicePayment",
		"SplitRule",
		"SmartContract",
		"DepositStake",
		"WithdrawStake",
		"DepositStakeTxV2",
		"StakeRewardDistributionTx",
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
		"Pending",
		"Valid",
		"Invalid",
		"Committed",
		"Directly Finalized",
		"Indirectly Finalized",
		"Trusted",
		"Disposed",
	}[s]
}
