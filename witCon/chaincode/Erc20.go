package chaincode

import (
	"math/big"
	"witCon/common"
	"witCon/common/block"
	"witCon/common/zerror"
)

const (
	NewAccount = "00000001"
	Transfer   = "00000002"
)

var (
	InitError     = zerror.New("当前新建账户已有余额", "Newly created accounts with existing balances", 1001)
	TransferError = zerror.New("当前账户余额不足", "Insufficient current account balance", 1002)
)

func ResolveTx(tx *block.Transaction, vm VM) error {
	method := tx.Data[:8]
	if string(method) == NewAccount {
		toAddress := common.BytesToAddress(tx.Data[8:28])
		initValue := big.NewInt(0).SetBytes(tx.Data[28:])
		lastValue := vm.ReadState(toAddress)
		if len(lastValue) != 0 {
			return InitError
		}
		vm.WriteState(toAddress, initValue.Bytes())
	} else {

		from := tx.From
		toAddress := common.BytesToAddress(tx.Data[8:28])
		amount := big.NewInt(0).SetBytes(tx.Data[28:])

		//log.Debug("tx", "number", tx.Number, "from", tx.From.String(), "data", hexutil.Encode(tx.Data), "to", toAddress, "value", amount)
		_fromBalance := vm.ReadState(from)
		fromBalance := big.NewInt(0).SetBytes(_fromBalance)
		if _fromBalance == nil {
			fromBalance, _ = big.NewInt(0).SetString("100000000000000000000", 10)
			//log.Debug("pre value", "value", hexutil.Encode(fromBalance.Bytes()))
		}
		result := fromBalance.Sub(fromBalance, amount)
		vm.WriteState(from, result.Bytes())

		_toBalance := vm.ReadState(toAddress)
		toBalance := big.NewInt(0).SetBytes(_toBalance)
		result = toBalance.Add(toBalance, amount)
		vm.WriteState(toAddress, result.Bytes())
	}
	return nil
}
