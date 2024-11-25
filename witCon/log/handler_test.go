package log

import (
	"strconv"
	"strings"
	"testing"
)

func TestFreshLogFiles(t *testing.T) {
	//path := "D:\\A-blockchain\\project\\zlattice\\data"
	//freshLogFiles(path)

	//goid不能超过int范围，超出会出现溢出
	goid := strconv.Itoa(int(213232131231231312312242427))
	if len(strconv.Itoa(int(213232131231231312312242427))) < 6 {
		goid = strings.Repeat("0", 6-len(strconv.Itoa(int(213232131231231312312242427)))) + strconv.Itoa(int(213232131231231312312242427))
	}
	t.Log(goid)
}

//xxxx_xxx_2021-2-3.log
//xxxx_xxx_2021-2-3.txt
//xxxx.txt
//xxxx.log
//zltc_dd_2021-10-25.log
