package controllers

import (
	"go-trading-bot/app/models"
	"go-trading-bot/bitflyer"
	"go-trading-bot/config"
	"log"
)

// StreamIngestionData will pass data from bitflyer package to candle stick package
func StreamIngestionData() {
	c := config.Config
	ai := NewAI(c.ProductCode, c.TradeDuration, c.DataLimit, c.UsePercent, c.StopLimitPercent, c.BackTest)
	// new channel which contains each ticker
	var tickerChannel = make(chan bitflyer.Ticker)
	apiClient := bitflyer.New(config.Config.ApiKey, config.Config.ApiSecret)
	go apiClient.GetRealTimeTicker(config.Config.ProductCode, tickerChannel)
	// 1分間のテーブル、1秒のテーブルなどそれぞれに書き込むためのループ
	// go routineにすることでStream Dataを撮り続けつつ、UI描画したりできる
	go func() {
		for ticker := range tickerChannel {
			log.Printf("action=StreamIngestionData, %v", ticker)
			for _, duration := range config.Config.Durations { // Duratios: 1s, 1m, 1h
				isCreated := models.CreateCandleWithDuration(ticker, ticker.ProductCode, duration)
				// when the candle is newly created - we trade i.e., decide if this is the trade entry point
				// how often candle stick is created is depends on duration
				if isCreated == true && duration == config.Config.TradeDuration {

					// TODO
					ai.Trade()
				}
			}
		}
	}()
}
