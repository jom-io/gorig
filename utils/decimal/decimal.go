package decimal

import (
	"math"
	"math/big"
)

var (
	AmountPrecision = 2 // 金额精度 保留2位小数
)

func Round(val float64, precision int) float64 {
	p := math.Pow10(precision)
	return math.Round(val*p) / p
}

func Add(a, b float64) float64 {
	aF := big.NewFloat(a)
	bF := big.NewFloat(b)
	total := new(big.Float).Add(aF, bF)
	totalAmount, _ := total.Float64()
	return Round(totalAmount, AmountPrecision)
}

func Sub(a, b float64) float64 {
	aF := big.NewFloat(a)
	bF := big.NewFloat(b)
	total := new(big.Float).Sub(aF, bF)
	totalAmount, _ := total.Float64()
	return Round(totalAmount, AmountPrecision)
}

func Mul(a, b float64) float64 {
	aF := big.NewFloat(a)
	bF := big.NewFloat(b)
	total := new(big.Float).Mul(aF, bF)
	totalAmount, _ := total.Float64()
	return Round(totalAmount, AmountPrecision)
}

func Div(a, b float64) float64 {
	aF := big.NewFloat(a)
	bF := big.NewFloat(b)
	total := new(big.Float).Quo(aF, bF)
	totalAmount, _ := total.Float64()
	return Round(totalAmount, AmountPrecision)
}

func Equal(a, b float64) bool {
	return math.Abs(a-b) < 0.000001
}
