package models

import (
	"encoding/json"
	"fmt"
	"go-trading-bot/config"
	"log"
	"strings"
	"time"
)

// TradeSignalEvent => use when trade is executed
type TradeSignalEvent struct {
	Time        time.Time `json:"time"`
	ProductCode string    `json:"product_code"`
	Side        string    `json:"side"`
	Price       float64   `json:"price"`
	Size        float64   `json:"size"`
}

// Save will return true if successfully insert data into the database
func (trade *TradeSignalEvent) Save() bool {
	cmd := fmt.Sprintf("INSERT INTO %s (time, product_code, side, price, size) VALUES (?, ?, ?, ?, ?)", tableNameSignalEvents)
	_, err := DbConnection.Exec(cmd, trade.Time.Format(time.RFC3339), trade.ProductCode, trade.Side, trade.Price, trade.Size)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Println(err)
			return true
		}
		return false
	}
	return true
}

// TradeSignalEvents that holds several signal struct.
type TradeSignalEvents struct {
	TradeSignals []TradeSignalEvent `json:"signals,omitempty"`
}

// NewTradeSignalEvents => constructor
func NewTradeSignalEvents() *TradeSignalEvents {
	return &TradeSignalEvents{}
}

// GetTradeSignalEventsByCount returns only specified number of latest trade result
func GetTradeSignalEventsByCount(loadEvents int) *TradeSignalEvents {
	cmd := fmt.Sprintf(`SELECT * FROM (
		SELECT time, product_code, side, price, size FROM %s WHERE product_code = ? ORDER BY time DESC LIMIT ? )
		ORDER BY time ASC;`, tableNameSignalEvents)
	rows, err := DbConnection.Query(cmd, config.Config.ProductCode, loadEvents)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var tradeSignalEvents TradeSignalEvents
	for rows.Next() {
		var tradeSignalEvent TradeSignalEvent
		rows.Scan(&tradeSignalEvent.Time, &tradeSignalEvent.ProductCode, &tradeSignalEvent.Side, &tradeSignalEvent.Price, &tradeSignalEvent.Size)
		tradeSignalEvents.TradeSignals = append(tradeSignalEvents.TradeSignals, tradeSignalEvent)
	}
	err = rows.Err()
	if err != nil {
		return nil
	}
	return &tradeSignalEvents
}

// GetTradeSignalEventsAfterTime returns trade data after specified time
func GetTradeSignalEventsAfterTime(getTime time.Time) *TradeSignalEvents {
	cmd := fmt.Sprintf(`SELECT * FROM (
		SELECT time, product_code, side, price, size FROM %s
		WHERE DATETIME(time) >= DATETIME(?)
		ORDER BY time DESC
) ORDER BY time ASC;`, tableNameSignalEvents)
	rows, err := DbConnection.Query(cmd, getTime.Format(time.RFC3339))
	if err != nil {
		return nil
	}
	defer rows.Close()

	var tradeSignalEvents TradeSignalEvents
	for rows.Next() {
		var tradeSignalEvent TradeSignalEvent
		rows.Scan(&tradeSignalEvent.Time, &tradeSignalEvent.ProductCode, &tradeSignalEvent.Side, &tradeSignalEvent.Price, &tradeSignalEvent.Size)
		tradeSignalEvents.TradeSignals = append(tradeSignalEvents.TradeSignals, tradeSignalEvent)
	}
	err = rows.Err()
	if err != nil {
		return nil
	}
	return &tradeSignalEvents
}

func (trade *TradeSignalEvents) CanBuy(time time.Time) bool {
	lenTradeSignals := len(trade.TradeSignals)
	if lenTradeSignals == 0 {
		return true
	}

	lastTradeSignal := trade.TradeSignals[lenTradeSignals-1]
	if lastTradeSignal.Side == "SELL" && lastTradeSignal.Time.Before(time) {
		return true
	}
	return false

}

func (trade *TradeSignalEvents) CanSell(time time.Time) bool {
	lenTradeSignals := len(trade.TradeSignals)
	if lenTradeSignals == 0 {
		return false
	}

	lastTradeSignal := trade.TradeSignals[lenTradeSignals-1]
	if lastTradeSignal.Side == "BUY" && lastTradeSignal.Time.Before(time) {
		return true
	}
	return false

}

func (trade *TradeSignalEvents) Buy(ProductCode string, time time.Time, price, size float64, save bool) bool {
	if !trade.CanBuy(time) {
		return false
	}
	buySignal := TradeSignalEvent{
		ProductCode: ProductCode,
		Time:        time,
		Side:        "BUY",
		Price:       price,
		Size:        size,
	}
	// if it is not backtest
	if save {
		buySignal.Save()
	}
	trade.TradeSignals = append(trade.TradeSignals, buySignal)
	return true
}

func (trade *TradeSignalEvents) Sell(ProductCode string, time time.Time, price, size float64, save bool) bool {
	if !trade.CanSell(time) {
		return false
	}
	sellSignal := TradeSignalEvent{
		ProductCode: ProductCode,
		Time:        time,
		Side:        "SELL",
		Price:       price,
		Size:        size,
	}
	// if it is not backtest
	if save {
		sellSignal.Save()
	}
	trade.TradeSignals = append(trade.TradeSignals, sellSignal)
	return true
}

func (trade *TradeSignalEvents) Profit() float64 {
	total := 0.0
	beforeSell := 0.0
	isHolding := false
	for i, signalTradeEvent := range trade.TradeSignals {
		if i == 0 && signalTradeEvent.Side == "SELL" {
			continue
		}
		if signalTradeEvent.Side == "BUY" {
			total -= signalTradeEvent.Price * signalTradeEvent.Size
			isHolding = true
		}
		if signalTradeEvent.Side == "SELL" {
			total += signalTradeEvent.Price * signalTradeEvent.Size
			isHolding = false
			beforeSell = total
		}
	}
	if isHolding == true {
		return beforeSell
	}
	return total
}

func (trade TradeSignalEvents) MarshalJSON() ([]byte, error) {
	value, err := json.Marshal(&struct {
		AllSignals []TradeSignalEvent `json:"signals,omitempty"`
		Profit     float64            `json:"profit,omitempty"`
	}{
		AllSignals: trade.TradeSignals,
		Profit:     trade.Profit(),
	})
	if err != nil {
		return nil, err
	}
	return value, err
}

func (trade *TradeSignalEvents) GetAfter(time time.Time) *TradeSignalEvents {
	for i, signal := range trade.TradeSignals {
		if time.After(signal.Time) {
			continue
		}
		return &TradeSignalEvents{TradeSignals: trade.TradeSignals[i:]}
	}
	return nil
}
