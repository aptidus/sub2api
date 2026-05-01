package payment

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

const centsPerYuan = 100
const maxYuanDecimalPlaces = 2

// ValidateYuanAmountFloat returns an error when the amount is invalid or has
// more than 2 decimal places.
func ValidateYuanAmountFloat(yuan float64) error {
	if math.IsNaN(yuan) || math.IsInf(yuan, 0) {
		return fmt.Errorf("invalid amount")
	}
	d := decimal.NewFromFloat(yuan)
	if d.Round(maxYuanDecimalPlaces).Equal(d) {
		return nil
	}
	return fmt.Errorf("amount must have at most 2 decimal places")
}

// YuanToFen converts a CNY yuan string (e.g. "10.50") to fen (int64).
// Uses shopspring/decimal for precision.
func YuanToFen(yuanStr string) (int64, error) {
	d, err := decimal.NewFromString(yuanStr)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", yuanStr)
	}
	if !d.Round(maxYuanDecimalPlaces).Equal(d) {
		return 0, fmt.Errorf("amount must have at most 2 decimal places: %s", yuanStr)
	}
	return d.Mul(decimal.NewFromInt(centsPerYuan)).IntPart(), nil
}

// FenToYuan converts fen (int64) to yuan as a float64 for interface compatibility.
func FenToYuan(fen int64) float64 {
	return decimal.NewFromInt(fen).Div(decimal.NewFromInt(centsPerYuan)).InexactFloat64()
}
