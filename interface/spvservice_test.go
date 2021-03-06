package _interface

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"os"
	"testing"

	tx "github.com/elastos/Elastos.ELA.SPV/core/transaction"
	"github.com/elastos/Elastos.ELA.SPV/log"
	"github.com/elastos/Elastos.ELA.SPV/spvwallet/config"
)

var spv SPVService

func TestNewSPVService(t *testing.T) {
	log.Init()

	var id = make([]byte, 8)
	var clientId uint64
	var err error
	rand.Read(id)
	binary.Read(bytes.NewReader(id), binary.LittleEndian, clientId)
	spv = NewSPVService(clientId, config.Values().SeedList)

	// Register account
	err = spv.RegisterAccount("ETBBrgotZy3993o9bH75KxjLDgQxBCib6u")
	err = spv.RegisterAccount("EUyNwnAh5SzzTtAPV1HkXzjUEbw2YqKsUM")
	if err != nil {
		t.Error("Register account error: ", err)
		os.Exit(0)
	}

	// Set on transaction confirmed callback
	spv.RegisterTransactionListener(&ConfirmedListener{txType: tx.TransferAsset})
	spv.RegisterTransactionListener(&UnconfirmedListener{txType: tx.TransferAsset})

	// Start spv service
	err = spv.Start()
	if err != nil {
		t.Error("Start SPV service error: ", err)
	}
}

type ConfirmedListener struct {
	txType tx.TransactionType
}

func (l *ConfirmedListener) Type() tx.TransactionType {
	return l.txType
}

func (l *ConfirmedListener) Confirmed() bool {
	return true
}

func (l *ConfirmedListener) Notify(proof Proof, tx tx.Transaction) {
	log.Debug("Receive confirmed transaction hash:", tx.Hash().String())
	err := spv.VerifyTransaction(proof, tx)
	if err != nil {
		log.Error("Verify transaction error: ", err)
		return
	}

	// Submit transaction receipt
	spv.SubmitTransactionReceipt(*tx.Hash())
}

type UnconfirmedListener struct {
	txType tx.TransactionType
}

func (l *UnconfirmedListener) Type() tx.TransactionType {
	return l.txType
}

func (l *UnconfirmedListener) Confirmed() bool {
	return false
}

func (l *UnconfirmedListener) Notify(proof Proof, tx tx.Transaction) {
	log.Debug("Receive unconfirmed transaction hash:", tx.Hash().String())
	err := spv.VerifyTransaction(proof, tx)
	if err != nil {
		log.Error("Verify transaction error: ", err)
		return
	}

	// Submit transaction receipt
	spv.SubmitTransactionReceipt(*tx.Hash())
}
