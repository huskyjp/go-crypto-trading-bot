package tradingalgo

import (
	"math"

	"github.com/markcheno/go-talib"
)

// HistoricalVolatility returns standard deviation depends on daily price change
// if SD is higher, the price move is meaningful but if SD is smaller, the price move is steady
func HistoricalVolatility(inReal []float64, inTimePeriod int) []float64 {
	change := make([]float64, 0)
	for i := range inReal {
		if i == 0 {
			continue
		}
		// calculate change ratio of Day n & Day n-1; log(n/n-1)
		dayChange := math.Log(
			float64(inReal[i]) / float64(inReal[i-1]))
		change = append(change, dayChange)
	}
	// calculate SD by passing (change ratio, period, daily percentage)
	// look at Day n & Day n-1 change, and check that Standard Deviation
	return talib.StdDev(change, inTimePeriod, math.Sqrt(1)*100)
}
