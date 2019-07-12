package quota

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"
	"testing"
)

func TestCalc(t *testing.T) {
	tmp := big.NewInt(987080173288357376)
	d := big.NewInt(67108863)
	d.Mul(d, qcDivision)
	d.Div(d, tmp)
	d.Add(d, helper.Big1)
	fmt.Println(d)
}
