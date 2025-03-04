package decimal

import (
	"math"
	"math/big"
)

const (
	AmountPrecision = 2 // 金额精度 保留2位小数
)

type Decimal struct {
	precision int
}

func New(precision int) *Decimal {
	return &Decimal{precision: precision}
}

func (d *Decimal) Round(val float64) float64 {
	return Round(val, d.precision)
}

func (d *Decimal) Add(a, b float64) float64 {
	return Add(a, b, d.precision)
}

func (d *Decimal) Sub(a, b float64) float64 {
	return Sub(a, b, d.precision)
}

func (d *Decimal) Mul(a, b float64) float64 {
	return Mul(a, b, d.precision)
}

func (d *Decimal) Div(a, b float64) float64 {
	return Div(a, b, d.precision)
}

func Round(val float64, use ...int) float64 {
	precision := AmountPrecision
	if len(use) > 0 {
		precision = use[0]
	}
	p := math.Pow10(precision)
	return math.Round(val*p) / p
}

func Add(a, b float64, percision ...int) float64 {
	aF := big.NewFloat(a)
	bF := big.NewFloat(b)
	total := new(big.Float).Add(aF, bF)
	totalAmount, _ := total.Float64()
	return Round(totalAmount, percision...)
}

func Sub(a, b float64, percision ...int) float64 {
	aF := big.NewFloat(a)
	bF := big.NewFloat(b)
	total := new(big.Float).Sub(aF, bF)
	totalAmount, _ := total.Float64()
	return Round(totalAmount, percision...)
}

func Mul(a, b float64, percision ...int) float64 {
	aF := big.NewFloat(a)
	bF := big.NewFloat(b)
	total := new(big.Float).Mul(aF, bF)
	totalAmount, _ := total.Float64()

	return Round(totalAmount, percision...)
}

func Div(a, b float64, percision ...int) float64 {
	aF := big.NewFloat(a)
	bF := big.NewFloat(b)
	total := new(big.Float).Quo(aF, bF)
	totalAmount, _ := total.Float64()
	return Round(totalAmount, percision...)
}

func Equal(a, b float64) bool {
	return math.Abs(a-b) < 0.000001
}
