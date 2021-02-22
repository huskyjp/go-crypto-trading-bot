package models

// decandle.go can return specific target value from Candles such as Candles.High etc...

import (
	"go-trading-bot/tradingalgo"
	"time"

	"github.com/markcheno/go-talib"
)

// DataFrameCandle struch which contains all information we are going to manipulate in this file
type DataFrameCandle struct {
	ProductCode   string             `json:"product_code"`
	Duration      time.Duration      `json:"duration"`
	Candles       []Candle           `json:"candles"`
	SMAs          []SMA              `json:"smas,omitempty"`
	EMAs          []EMA              `json:"emas,omitempty"`
	BBands        *BBands            `json:"bbands,omitempty"`
	IchimokuCloud *IchimokuCloud     `json:"ichimoku,omitempty"`
	Rsi           *Rsi               `json:"rsi,omitempty"`
	Macd          *Macd              `json:"macd,omitempty"`
	Hvs           []Hv               `json:"hvs,omitempty"`
	Events        *TradeSignalEvents `json:"events,omitempty"`
}

func (df *DataFrameCandle) AddEvents(timeTime time.Time) bool {
	tradeSignalEvents := GetTradeSignalEventsAfterTime(timeTime)
	if len(tradeSignalEvents.TradeSignals) > 0 {
		df.Events = tradeSignalEvents
		return true
	}
	return false
}

func (df *DataFrameCandle) BackTestEma(period1, period2 int) *TradeSignalEvents {
	lenCandles := len(df.Candles)
	if lenCandles <= period1 || lenCandles <= period2 {
		return nil
	}
	tradeSignalEvents := NewTradeSignalEvents()
	emaValue1 := talib.Ema(df.Closes(), period1)
	emaValue2 := talib.Ema(df.Closes(), period2)

	for i := 1; i < lenCandles; i++ {
		if i < period1 || i < period2 {
			continue
		}
		// golden cross
		if emaValue1[i-1] < emaValue2[i-1] && emaValue1[i] >= emaValue2[i] {
			tradeSignalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}
		// dead cross
		if emaValue1[i-1] > emaValue2[i-1] && emaValue1[i] <= emaValue2[i] {
			tradeSignalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}

	}
	return tradeSignalEvents
}

func (df *DataFrameCandle) OptimizeEma() (performance float64, bestPeriod1 int, bestPeriod2 int) {
	bestPeriod1 = 7
	bestPeriod2 = 14

	for period1 := 5; period1 < 50; period1++ {
		for period2 := 12; period2 < 50; period2++ {
			signalEvents := df.BackTestEma(period1, period2)
			if signalEvents == nil {
				continue
			}
			profit := signalEvents.Profit()
			if performance < profit {
				performance = profit
				bestPeriod1 = period1
				bestPeriod2 = period2
			}
		}
	}
	return performance, bestPeriod1, bestPeriod2
}

func (df *DataFrameCandle) BackTestBb(n int, k float64) *TradeSignalEvents {
	lenCandles := len(df.Candles)

	if lenCandles <= n {
		return nil
	}

	signalEvents := &TradeSignalEvents{}
	bbUp, _, bbDown := talib.BBands(df.Closes(), n, k, k, 0)
	for i := 1; i < lenCandles; i++ {
		if i < n {
			continue
		}
		if bbDown[i-1] > df.Candles[i-1].Close && bbDown[i] <= df.Candles[i].Close {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}
		if bbUp[i-1] < df.Candles[i-1].Close && bbUp[i] >= df.Candles[i].Close {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}
	}
	return signalEvents
}

func (df *DataFrameCandle) OptimizeBb() (performance float64, bestN int, bestK float64) {
	bestN = 20
	bestK = 2.0

	for n := 10; n < 20; n++ {
		for k := 1.9; k < 2.1; k += 0.1 {
			tradeSignalEvents := df.BackTestBb(n, k)
			if tradeSignalEvents == nil {
				continue
			}
			profit := tradeSignalEvents.Profit()
			if performance < profit {
				performance = profit
				bestN = n
				bestK = k
			}
		}
	}
	return performance, bestN, bestK
}

func (df *DataFrameCandle) BackTestIchimoku() *TradeSignalEvents {
	lenCandles := len(df.Candles)

	if lenCandles <= 52 {
		return nil
	}

	var signalEvents TradeSignalEvents
	tenkan, kijun, senkouA, senkouB, chikou := tradingalgo.IchimokuCloud(df.Closes())

	for i := 1; i < lenCandles; i++ {

		if chikou[i-1] < df.Candles[i-1].High && chikou[i] >= df.Candles[i].High &&
			senkouA[i] < df.Candles[i].Low && senkouB[i] < df.Candles[i].Low &&
			tenkan[i] > kijun[i] {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}

		if chikou[i-1] > df.Candles[i-1].Low && chikou[i] <= df.Candles[i].Low &&
			senkouA[i] > df.Candles[i].High && senkouB[i] > df.Candles[i].High &&
			tenkan[i] < kijun[i] {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}
	}
	return &signalEvents
}

func (df *DataFrameCandle) OptimizeIchimoku() (performance float64) {
	signalEvents := df.BackTestIchimoku()
	if signalEvents == nil {
		return 0.0
	}
	performance = signalEvents.Profit()
	return performance
}

func (df *DataFrameCandle) BackTestMacd(macdFastPeriod, macdSlowPeriod, macdSignalPeriod int) *TradeSignalEvents {
	lenCandles := len(df.Candles)

	if lenCandles <= macdFastPeriod || lenCandles <= macdSlowPeriod || lenCandles <= macdSignalPeriod {
		return nil
	}

	signalEvents := &TradeSignalEvents{}
	outMACD, outMACDSignal, _ := talib.Macd(df.Closes(), macdFastPeriod, macdSlowPeriod, macdSignalPeriod)

	for i := 1; i < lenCandles; i++ {
		if outMACD[i] < 0 &&
			outMACDSignal[i] < 0 &&
			outMACD[i-1] < outMACDSignal[i-1] &&
			outMACD[i] >= outMACDSignal[i] {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}

		if outMACD[i] > 0 &&
			outMACDSignal[i] > 0 &&
			outMACD[i-1] > outMACDSignal[i-1] &&
			outMACD[i] <= outMACDSignal[i] {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}
	}
	return signalEvents
}

func (df *DataFrameCandle) OptimizeMacd() (performance float64, bestMacdFastPeriod, bestMacdSlowPeriod, bestMacdSignalPeriod int) {
	bestMacdFastPeriod = 12
	bestMacdSlowPeriod = 26
	bestMacdSignalPeriod = 9

	for fastPeriod := 10; fastPeriod < 19; fastPeriod++ {
		for slowPeriod := 20; slowPeriod < 30; slowPeriod++ {
			for signalPeriod := 5; signalPeriod < 15; signalPeriod++ {
				signalEvents := df.BackTestMacd(fastPeriod, slowPeriod, signalPeriod)
				if signalEvents == nil {
					continue
				}
				profit := signalEvents.Profit()
				if performance < profit {
					performance = profit
					bestMacdFastPeriod = fastPeriod
					bestMacdSlowPeriod = slowPeriod
					bestMacdSignalPeriod = signalPeriod
				}
			}
		}
	}
	return performance, bestMacdFastPeriod, bestMacdSlowPeriod, bestMacdSignalPeriod
}

func (df *DataFrameCandle) BackTestRsi(period int, buyThread, sellThread float64) *TradeSignalEvents {
	lenCandles := len(df.Candles)
	if lenCandles <= period {
		return nil
	}

	signalEvents := NewTradeSignalEvents()
	values := talib.Rsi(df.Closes(), period)
	for i := 1; i < lenCandles; i++ {
		if values[i-1] == 0 || values[i-1] == 100 {
			continue
		}
		if values[i-1] < buyThread && values[i] >= buyThread {
			signalEvents.Buy(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}

		if values[i-1] > sellThread && values[i] <= sellThread {
			signalEvents.Sell(df.ProductCode, df.Candles[i].Time, df.Candles[i].Close, 0.05, false)
		}
	}
	return signalEvents
}

func (df *DataFrameCandle) OptimizeRsi() (performance float64, bestPeriod int, bestBuyThread, bestSellThread float64) {
	bestPeriod = 14
	bestBuyThread, bestSellThread = 30.0, 70.0

	for period := 5; period < 25; period++ {
		signalEvents := df.BackTestRsi(period, bestBuyThread, bestSellThread)
		if signalEvents == nil {
			continue
		}
		profit := signalEvents.Profit()
		if performance < profit {
			performance = profit
			bestPeriod = period
			bestBuyThread = bestBuyThread
			bestSellThread = bestSellThread
		}
	}
	return performance, bestPeriod, bestBuyThread, bestSellThread
}

// SMA =>  Single Moving Average that calculates price trends with several periods
// period => how long?, Values => actual average price
type SMA struct {
	Period int       `json:"period,omitempty"`
	Values []float64 `json:"values,omitempty"`
}

// AddSma returns true if a current number of Candle is larget than current period
// then add Period and Values (from talib) to the struct
func (df *DataFrameCandle) AddSma(period int) bool {
	if len(df.Candles) > period {
		df.SMAs = append(df.SMAs, SMA{
			Period: period,
			Values: talib.Sma(df.Closes(), period),
		})
		return true
	}
	return false
}

// EMA => Exponential Moving Average; (... + Nx * 2) / (N + 1)
// period => how long?, Values => actual average price
type EMA struct {
	Period int       `json:"period,omitempty"`
	Values []float64 `json:"values,omitempty"`
}

// AddEma returns true if a current number of Candle is larger than current period
// then add Period and Values (from talib) to the struct
func (df *DataFrameCandle) AddEma(period int) bool {
	if len(df.Candles) > period {
		df.EMAs = append(df.EMAs, EMA{
			Period: period,
			Values: talib.Ema(df.Closes(), period),
		})
		return true
	}
	return false
}

// BBands => Bollinger Bands; relatioship between SMA & SD
// N => period, K => SD (2, contains 98%)
type BBands struct {
	N    int       `json:"n,omitempty"`
	K    float64   `json:"k,omitempty"`
	Up   []float64 `json:"up,omitempty"`
	Mid  []float64 `json:"mid,omitempty"`
	Down []float64 `json:"down,omitempty"`
}

// AddBBands => return true if the length of period is less than length of slice of Closes struct
// get up, mid, down; represents bollinger bands range
func (df *DataFrameCandle) AddBBands(n int, k float64) bool {
	if n <= len(df.Closes()) {
		up, mid, down := talib.BBands(df.Closes(), n, k, k, 0)
		df.BBands = &BBands{
			N:    n,
			K:    k,
			Up:   up,
			Mid:  mid,
			Down: down,
		}
		return true
	}
	return false
}

// IchimokuCloud => 5 lines price averages
// Tenakn => (9days Max + Min) / 2, Kijun => (26days max + min) / 2, SenkouA => (Tenkan + Kijyun) / 2 *26days ahead
// SenkouB => (52days Max + Min) / 2 * 26days ahead, Chikou => Today's Closing Price *26days behind
type IchimokuCloud struct {
	Tenkan  []float64 `json:"tenkan,omitempty"`
	Kijun   []float64 `json:"kijun,omitempty"`
	SenkouA []float64 `json:"senkoua,omitempty"`
	SenkouB []float64 `json:"senkoub,omitempty"`
	Chikou  []float64 `json:"chikou,omitempty"`
}

// AddIchimoku returns true if Close slice length is longer than 9
// then insert each value from alog.go
func (df *DataFrameCandle) AddIchimoku() bool {
	tenkanN := 9
	if len(df.Closes()) >= tenkanN {
		tenkan, kijun, senkouA, senkouB, chikou := tradingalgo.IchimokuCloud(df.Closes())
		df.IchimokuCloud = &IchimokuCloud{
			Tenkan:  tenkan,
			Kijun:   kijun,
			SenkouA: senkouA,
			SenkouB: senkouB,
			Chikou:  chikou,
		}
		return true
	}
	return false
}

// Rsi => length of the day and its values
type Rsi struct {
	Period int       `json:"period,omitempty"`
	Values []float64 `json:"values,omitempty"`
}

// AddRsi returns true if period is less than current candle length
func (df *DataFrameCandle) AddRsi(period int) bool {
	if len(df.Candles) > period {
		values := talib.Rsi(df.Closes(), period)
		df.Rsi = &Rsi{
			Period: period,
			Values: values,
		}
		return true
	}
	return false
}

// Macd => FastPeriod...short term EMA, SlowPeriod...long term EMA, SignalPeriod...MACD EMA (period less than fastperiod)
type Macd struct {
	FastPeriod   int       `json:"fast_period,omitempty"`
	SlowPeriod   int       `json:"slow_period,omitempty"`
	SignalPeriod int       `json:"signal_period,omitempty"`
	Macd         []float64 `json:"macd,omitempty"`
	MacdSignal   []float64 `json:"macd_signal,omitempty"`
	MacdHist     []float64 `json:"macd_hist,omitempty"`
}

// AddMacd returns true if current candle length is more than 1
func (df *DataFrameCandle) AddMacd(inFastPeriod, inSlowPeriod, inSignalPeriod int) bool {
	if len(df.Candles) > 1 {
		outMACD, outMACDSignal, outMACDHist := talib.Macd(df.Closes(), inFastPeriod, inSlowPeriod, inSignalPeriod)
		df.Macd = &Macd{
			FastPeriod:   inFastPeriod,
			SlowPeriod:   inSlowPeriod,
			SignalPeriod: inSignalPeriod,
			Macd:         outMACD,
			MacdSignal:   outMACDSignal,
			MacdHist:     outMACDHist,
		}
		return true
	}
	return false
}

type Hv struct {
	Period int       `json:"period,omitempty"`
	Values []float64 `json:"values,omitempty"`
}

func (df *DataFrameCandle) AddHv(period int) bool {
	if len(df.Candles) >= period {
		df.Hvs = append(df.Hvs, Hv{
			Period: period,
			Values: tradingalgo.HistoricalVolatility(df.Closes(), period),
		})
		return true
	}
	return false
}

// Times return slice only contains time value
func (df *DataFrameCandle) Times() []time.Time {
	// create slice that contains type of time & length of Candles
	timeSlice := make([]time.Time, len(df.Candles))
	for i, candle := range df.Candles {
		timeSlice[i] = candle.Time // store only each time from Candle
	}
	return timeSlice
}

// Opens return slice only contains open value
func (df *DataFrameCandle) Opens() []float64 {
	openSlice := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		openSlice[i] = candle.Open
	}
	return openSlice
}

// Closes return slice only contains close value
func (df *DataFrameCandle) Closes() []float64 {
	closeSlice := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		closeSlice[i] = candle.Close
	}
	return closeSlice
}

// Highs return slice only contains high value
func (df *DataFrameCandle) Highs() []float64 {
	highSlice := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		highSlice[i] = candle.High
	}
	return highSlice
}

// Lows return slice only contains low value
func (df *DataFrameCandle) Lows() []float64 {
	lowSlice := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		lowSlice[i] = candle.Low
	}
	return lowSlice
}

// Volumes return slice only contains volume value
func (df *DataFrameCandle) Volumes() []float64 {
	volumeSlice := make([]float64, len(df.Candles))
	for i, candle := range df.Candles {
		volumeSlice[i] = candle.Volume
	}
	return volumeSlice
}
