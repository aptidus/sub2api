package payment

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

// ValidateYuanAmountFloat returns an error when the amount is invalid or has
// more than 2 decimal places.
func ValidateYuanAmountFloat(yuan float64) error {
	if math.IsNaN(yuan) || math.IsInf(yuan, 0) {
		return fmt.Errorf("invalid amount")
	}
	d := decimal.NewFromFloat(yuan)
	if d.Round(2).Equal(d) {
		return nil
	}
	return fmt.Errorf("amount must have at most 2 decimal places")
}

func YuanToFen(yuanStr string) (int64, error) {
	return AmountToMinorUnit(yuanStr, DefaultPaymentCurrency)
}

func FenToYuan(fen int64) float64 {
	return MinorUnitToAmount(fen, DefaultPaymentCurrency)
}
