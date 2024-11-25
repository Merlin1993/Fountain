package zerror

import (
	"encoding/json"
	"fmt"
)

var (
	mLocal      = CN
	DefaultCN   = CN
	DefaultCode = 10000
)

const (
	NEXT_CODE = 4400
	CN        = iota
	EN
)

//设置本地语言，默认中文
func SetLocal(local int) {
	mLocal = local
}

type ErrorExhaustiveInterface interface {
	Error() string
	Code() int
}

func New(cn string, en string, code int) error {
	ee := &ErrorExhaustive{s: make(map[int]string), Code: code}
	ee.s[CN] = cn
	ee.s[EN] = en
	return ee
}

func (e *ErrorExhaustive) Append(local int, error string) {
	e.s[mLocal] = error
}

// errorString is a trivial implementation of error.
type ErrorExhaustive struct {
	s    map[int]string
	Code int
}

func (e *ErrorExhaustive) Error() string {
	s, ok := e.s[mLocal]
	if !ok {
		return e.s[DefaultCN]
	}
	return s
}

// errorString is a trivial implementation of error.
type errorExhaustiveJson struct {
	err  string
	code int
}

func (e *ErrorExhaustive) MarshalJSON() ([]byte, error) {
	eej := &errorExhaustiveJson{
		err:  e.Error(),
		code: e.Code,
	}
	return json.Marshal(eej)
}

type ErrorParams struct {
	str  map[int]string
	Code int
}

func NewErrorParams(cn string, en string, code int) *ErrorParams {
	ep := &ErrorParams{str: make(map[int]string), Code: code}
	ep.str[CN] = cn
	ep.str[EN] = en
	return ep
}

func (ep *ErrorParams) ErrorOf(args ...interface{}) *ErrorExhaustive {
	ee := &ErrorExhaustive{s: make(map[int]string), Code: ep.Code}
	ee.s[mLocal] = fmt.Sprintf(ep.str[mLocal], args...)
	return ee
}
