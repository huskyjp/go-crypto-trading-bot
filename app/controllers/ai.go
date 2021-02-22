package controllers

import (
	"fmt"
	"go-trading-bot/app/models"
	"go-trading-bot/bitflyer"
	"go-trading-bot/config"
	"go-trading-bot/tradingalgo"
	"log"
	"math"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/markcheno/go-talib"
)

const (
	ApiFeePercent = 0.15
)

// AI => stores all info to trade automatically
type AI struct {
	API                  *bitflyer.APIClient
	ProductCode          string
	CurrencyCode         string
	CoinCode             string
	UsePercent           float64
	MinuteToExpires      int
	Duration             time.Duration
	PastPeriod           int
	SignalEvents         *models.TradeSignalEvents
	OptimizedTradeParams *models.TradeParams
	TradeSemaphore       *semaphore.Weighted
	StopLimit            float64
	StopLimitPercent     float64
	BackTest             bool
	StartTrade           time.Time
}

// TODO mutex, singleton is ideal
var Ai *AI

// NewAI contstructs new AI Trade Base Model, returns *AI
func NewAI(productCode string, duration time.Duration, pastPeriod int, UsePercent, stopLimitPercent float64, backTest bool) *AI {
	// new api client
	apiClient := bitflyer.New(config.Config.ApiKey, config.Config.ApiSecret)
	// signal event struct
	var signalEvents *models.TradeSignalEvents
	// confirm if it is backtest
	if backTest {
		signalEvents = models.NewTradeSignalEvents()
	} else {
		signalEvents = models.GetTradeSignalEventsByCount(1)
	}
	// split BTC & USD
	codes := strings.Split(productCode, "_")
	Ai = &AI{
		API:              apiClient,
		ProductCode:      productCode,
		CoinCode:         codes[0],
		CurrencyCode:     codes[1],
		UsePercent:       UsePercent,
		MinuteToExpires:  1,
		PastPeriod:       pastPeriod,
		Duration:         duration,
		SignalEvents:     signalEvents,
		TradeSemaphore:   semaphore.NewWeighted(1), // restrict only one goroutine
		BackTest:         backTest,
		StartTrade:       time.Now(),
		StopLimitPercent: stopLimitPercent,
	}
	// optimize parameters
	Ai.UpdateOptimizeParams(false)
	return Ai
}

// UpdateOptimizeParams gets candle stick dataframe, and optimize the parameters
func (ai *AI) UpdateOptimizeParams(isContinue bool) {
	// get specified dataframe candle
	df, _ := models.GetAllCandle(ai.ProductCode, ai.Duration, ai.PastPeriod)
	// optimizer returns trade params such as EMA...
	ai.OptimizedTradeParams = df.OptimizeParams()
	log.Printf("optimized_trade_params=%+v", ai.OptimizedTradeParams)
	if ai.OptimizedTradeParams == nil && isContinue && !ai.BackTest {
		log.Print("status_no_params")
		time.Sleep(5 * ai.Duration)
		ai.UpdateOptimizeParams(isContinue)
	}
}

// Buy returns childOrderAccenptanceID/isOrderCompleted from apiClient when the buy order is executed successfully
func (ai *AI) Buy(candle models.Candle) (childOrderAcceptanceID string, isOrderCompleted bool) {
	// check if backtest is true
	if ai.BackTest {
		couldBuy := ai.SignalEvents.Buy(ai.ProductCode, candle.Time, candle.Close, 1.0, false)
		fmt.Println("just buy")
		return "", couldBuy
	}

	//TODO
	if ai.StartTrade.After(candle.Time) {
		return
	}

	if !ai.SignalEvents.CanBuy(candle.Time) {
		return
	}

	availableCurrency, _ := ai.GetAvailableBalance()
	useCurrency := availableCurrency * ai.UsePercent
	ticker, err := ai.API.GetTicker(ai.ProductCode)
	if err != nil {
		return
	}
	size := 1 / (ticker.BestAsk / useCurrency)
	size = ai.AdjustSize(size)

	order := &bitflyer.Order{
		ProductCode:     ai.ProductCode,
		ChildOrderType:  "MARKET",
		Side:            "BUY",
		Size:            size,
		MinuteToExpires: ai.MinuteToExpires,
		TimeInForce:     "GTC",
	}
	log.Printf("status=order candle=%+v order=%+v", candle, order)
	resp, err := ai.API.SendOrder(order)
	if err != nil {
		log.Println(err)
		return
	}
	childOrderAcceptanceID = resp.ChildOrderAcceptanceID
	if resp.ChildOrderAcceptanceID == "" {
		// Insufficient fund
		log.Printf("order=%+v status=no_id", order)
		return
	}

	isOrderCompleted = ai.WaitUntilOrderComplete(childOrderAcceptanceID, candle.Time)
	fmt.Println("JUST BOUGHT: ")
	return childOrderAcceptanceID, isOrderCompleted
}

// Sell returns childOrderAccenptanceID/isOrderCompleted from apiClient when the sell order is executed successfully
func (ai *AI) Sell(candle models.Candle) (childOrderAcceptanceID string, isOrderCompleted bool) {
	if ai.BackTest {
		couldSell := ai.SignalEvents.Sell(ai.ProductCode, candle.Time, candle.Close, 1.0, false)
		fmt.Println("just sell")
		return "", couldSell
	}

	// TODO
	if ai.StartTrade.After(candle.Time) {
		return
	}

	if !ai.SignalEvents.CanSell(candle.Time) {
		return
	}

	_, availableCoin := ai.GetAvailableBalance()
	size := ai.AdjustSize(availableCoin)
	order := &bitflyer.Order{
		ProductCode:     ai.ProductCode,
		ChildOrderType:  "MARKET",
		Side:            "SELL",
		Size:            size,
		MinuteToExpires: ai.MinuteToExpires,
		TimeInForce:     "GTC",
	}
	log.Printf("status=sell candle=%+v order=%+v", candle, order)
	resp, err := ai.API.SendOrder(order)
	if err != nil {
		log.Println(err)
		return
	}
	if resp.ChildOrderAcceptanceID == "" {
		// Insufficient funds
		log.Printf("order=%+v status=no_id", order)
		return
	}
	childOrderAcceptanceID = resp.ChildOrderAcceptanceID
	isOrderCompleted = ai.WaitUntilOrderComplete(childOrderAcceptanceID, candle.Time)
	return childOrderAcceptanceID, isOrderCompleted
}

// Trade
func (ai *AI) Trade() {
	isAcquire := ai.TradeSemaphore.TryAcquire(1)
	if !isAcquire {
		log.Println("Could not get trade lock")
		return
	}
	defer ai.TradeSemaphore.Release(1)
	// get optimized trade parameter such as EMA...
	params := ai.OptimizedTradeParams
	if params == nil {
		return
	}
	// get current candle stick dataframe
	df, _ := models.GetAllCandle(ai.ProductCode, ai.Duration, ai.PastPeriod)
	// length of the candles
	lenCandles := len(df.Candles)

	// EMA
	var emaValues1 []float64
	var emaValues2 []float64
	if params.EmaEnable {
		emaValues1 = talib.Ema(df.Closes(), params.EmaPeriod1)
		emaValues2 = talib.Ema(df.Closes(), params.EmaPeriod2)
	}

	// Bolinger Bands
	var bbUp []float64
	var bbDown []float64
	if params.BbEnable {
		bbUp, _, bbDown = talib.BBands(df.Closes(), params.BbN, params.BbK, params.BbK, 0)
	}

	// Ichimoku
	var tenkan, kijun, senkouA, senkouB, chikou []float64
	if params.IchimokuEnable {
		tenkan, kijun, senkouA, senkouB, chikou = tradingalgo.IchimokuCloud(df.Closes())
	}

	// MACD
	var outMACD, outMACDSignal []float64
	if params.MacdEnable {
		outMACD, outMACDSignal, _ = talib.Macd(df.Closes(), params.MacdFastPeriod, params.MacdSlowPeriod, params.MacdSignalPeriod)
	}

	// RSI
	var rsiValues []float64
	if params.RsiEnable {
		rsiValues = talib.Rsi(df.Closes(), params.RsiPeriod)
	}

	// Algorithm that find buypoint and sellpoint
	for i := 1; i < lenCandles; i++ {
		// we buy & sell when at least 3 of the algo say YES
		buyPoint, sellPoint := 0, 0
		// Golden Cross: EMA enable?EMAPeriod is less than i?
		if params.EmaEnable && params.EmaPeriod1 <= i && params.EmaPeriod2 <= i && df.Volumes()[i] > 100 {
			// check if it is Golden Cross
			if emaValues1[i-1] < emaValues2[i-1] && emaValues1[i] >= emaValues2[i] {
				buyPoint++
			}
			// Dead Cross??
			if emaValues1[i-1] > emaValues2[i-1] && emaValues1[i] <= emaValues2[i] {
				sellPoint++
			}
		}

		// Borinder Bands
		if params.BbEnable && params.BbN <= i {
			// Buy when below band
			if bbDown[i-1] > df.Candles[i-1].Close && bbDown[i] <= df.Candles[i].Close && df.Volumes()[i] > 100 {
				buyPoint++
			}
			// Sell when upper band
			if bbUp[i-1] < df.Candles[i-1].Close && bbUp[i] >= df.Candles[i].Close {
				sellPoint++
			}
		}

		// MACD
		if params.MacdEnable {
			if outMACD[i] < 0 && outMACDSignal[i] < 0 && outMACD[i-1] < outMACDSignal[i-1] && outMACD[i] >= outMACDSignal[i] && df.Volumes()[i] > 100 {
				buyPoint++
			}

			if outMACD[i] > 0 && outMACDSignal[i] > 0 && outMACD[i-1] > outMACDSignal[i-1] && outMACD[i] <= outMACDSignal[i] {
				sellPoint++
			}
		}

		// Ichimoku
		if params.IchimokuEnable {
			if chikou[i-1] < df.Candles[i-1].High && chikou[i] >= df.Candles[i].High &&
				senkouA[i] < df.Candles[i].Low && senkouB[i] < df.Candles[i].Low &&
				tenkan[i] > kijun[i] && df.Volumes()[i] > 100 {
				buyPoint++
			}

			if chikou[i-1] > df.Candles[i-1].Low && chikou[i] <= df.Candles[i].Low &&
				senkouA[i] > df.Candles[i].High && senkouB[i] > df.Candles[i].High &&
				tenkan[i] < kijun[i] {
				sellPoint++
			}
		}

		// RSI
		if params.RsiEnable && rsiValues[i-1] != 0 && rsiValues[i-1] != 100 && df.Volumes()[i] > 100 {
			if rsiValues[i-1] < params.RsiBuyThread && rsiValues[i] >= params.RsiBuyThread {
				buyPoint++
			}

			if rsiValues[i-1] > params.RsiSellThread && rsiValues[i] <= params.RsiSellThread {
				sellPoint++
			}
		}

		// BUY when 3 algo says yes
		if buyPoint > 1 {
			_, isOrderCompleted := ai.Buy(df.Candles[i])
			if !isOrderCompleted {
				continue
			}
			// assign stop limit when ALGO was not correct... (this time 90%, i.e, 10& losscut)
			ai.StopLimit = df.Candles[i].Close * ai.StopLimitPercent
		}

		// SELl when 3 algo says yes
		if sellPoint > 1 || ai.StopLimit > df.Candles[i].Close {
			_, isOrderCompleted := ai.Sell(df.Candles[i])
			if !isOrderCompleted {
				continue
			}
			ai.StopLimit = 0.0
			// Optimize Params AGAIN after BUY=>SELL routine since the market is changed during one trade
			// optimize is always runing backend goroutine
			go ai.UpdateOptimizeParams(true)
		}
	}
}

func (ai *AI) GetAvailableBalance() (availableCurrency, availableCoin float64) {
	balances, err := ai.API.GetBalance()
	if err != nil {
		return
	}
	for _, balance := range balances {
		if balance.CurrentCode == ai.CurrencyCode {
			availableCurrency = balance.Available
		} else if balance.CurrentCode == ai.CoinCode {
			availableCoin = balance.Available
		}
	}
	return availableCurrency, availableCoin
}

func (ai *AI) AdjustSize(size float64) float64 {
	fee := size * ApiFeePercent
	size = size - fee
	return math.Floor(size*10000) / 10000
}

func (ai *AI) WaitUntilOrderComplete(childOrderAcceptanceID string, executeTime time.Time) bool {
	params := map[string]string{
		"product_code":              ai.ProductCode,
		"child_order_acceptance_id": childOrderAcceptanceID,
	}
	expire := time.After(time.Minute + (20 * time.Second))
	interval := time.Tick(15 * time.Second)
	return func() bool {
		for {
			select {
			case <-expire:
				return false
			case <-interval:
				listOrders, err := ai.API.ListOrder(params)
				if err != nil {
					return false
				}
				if len(listOrders) == 0 {
					return false
				}
				order := listOrders[0]
				if order.ChildOrderState == "COMPLETED" {
					if order.Side == "BUY" {
						couldBuy := ai.SignalEvents.Buy(ai.ProductCode, executeTime, order.AveragePrice, order.Size, true)
						if !couldBuy {
							log.Printf("status=buy childOrderAcceptanceID=%s order=%+v", childOrderAcceptanceID, order)
						}
						return couldBuy
					}
					if order.Side == "SELL" {
						couldSell := ai.SignalEvents.Sell(ai.ProductCode, executeTime, order.AveragePrice, order.Size, true)
						if !couldSell {
							log.Printf("status=sell childOrderAcceptanceID=%s order=%+v", childOrderAcceptanceID, order)
						}
						return couldSell
					}
					return false
				}
			}
		}
	}()
}
