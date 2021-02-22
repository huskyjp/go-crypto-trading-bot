package tradingalgo

// minMax returns 2 values which represents minimum price & max price
func minMax(inReal []float64) (float64, float64) {
	// insert temp at index0
	min := inReal[0]
	max := inReal[0]
	for _, price := range inReal {
		if min > price {
			min = price
		}
		if max < price {
			max = price
		}
	}
	return min, max
}

// get minimum
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

/*
Tenkan = (9-day high + 9-day low) / 2
Kijun = (26-day high + 26-day low) / 2
Senkou Span A = (Tenkan + Kijun) / 2 => 26days ahead
Senkou Span B = (52-day high + 52-day low) / 2 => 26days ahead
Chikou Span = Close plotted 26 days in the past
*/

// IchimokuCloud returns 5 float64 value that represents ichimoku shown in above
//
func IchimokuCloud(inReal []float64) ([]float64, []float64, []float64, []float64, []float64) {
	length := len(inReal)
	// create slice depends on the current length since we don't necessarily loop default value if
	// the length is less than the one
	tenkan := make([]float64, min(9, length))
	kijun := make([]float64, min(26, length))
	senkouA := make([]float64, min(26, length))
	senkouB := make([]float64, min(52, length))
	chikou := make([]float64, min(26, length))

	for i := range inReal {
		if i >= 9 {
			min, max := minMax(inReal[i-9 : i])
			tenkan = append(tenkan, (min+max)/2)
		}
		if i >= 26 {
			min, max := minMax(inReal[i-26 : i])
			kijun = append(kijun, (min+max)/2)
			senkouA = append(senkouA, (tenkan[i]+kijun[i])/2)
			chikou = append(chikou, inReal[i-26])
		}

		if i >= 52 {
			min, max := minMax(inReal[i-52 : i])
			senkouB = append(senkouB, (min+max)/2)
		}
	}
	return tenkan, kijun, senkouA, senkouB, chikou
}
