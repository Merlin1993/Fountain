package hexutil

import (
	"fmt"
	"witCon/common/zerror"
)

var (
	ErrEmptyString         = &decError{zerror.New("hex编码为空.", "empty hex string", 4100)}
	ErrSyntax              = &decError{zerror.New("hex编码错误.", "invalid hex string", 4101)}
	ErrMissingPrefix       = &decError{zerror.New("hex编码未以0x开头.", "hex string without 0x prefix", 4102)}
	ErrOddLength           = &decError{zerror.New("hex编码有奇数位个，应为偶数位个.", "hex string of odd length", 4103)}
	ErrEmptyNumber         = &decError{zerror.New("hex编码空时应为\"0x\".", "hex string \"0x\"", 4104)}
	ErrLeadingZero         = &decError{zerror.New("hex编码前导为0.", "hex number with leading zero digits", 4105)}
	ErrUint64Range         = &decError{zerror.New("hex编码应该为64bit.", "hex number > 64 bits", 4106)}
	ErrUintRange           = &decError{zerror.New(fmt.Sprintf("hex编码应该为%dbits.", uintBits), fmt.Sprintf("hex number > %d bits", uintBits), 4107)}
	ErrBig256Range         = &decError{zerror.New("hex编码应该为256bit.", "hex number > 256 bits", 4108)}
	ErrHexLengthFail       = zerror.NewErrorParams("%s需要hex编码格式长度为%d,当前长度为%d.", "%s want %d on hex length, but length is %d, ", 4109)
	ErrUnmarshallNonString = zerror.New("非字符串格式", " non-string ", 4110)
	DefaultCode            = 4111
)

type decError struct{ msg error }

func (err decError) Error() string { return err.msg.Error() }

func (err decError) Code() int {
	if err, ok := err.msg.(*zerror.ErrorExhaustive); ok {
		return err.Code
	} else {
		return DefaultCode
	}
}
