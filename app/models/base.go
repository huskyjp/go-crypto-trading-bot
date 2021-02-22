package models

import (
	"database/sql"
	"fmt"
	"go-trading-bot/config"
	"log"
	"time"

	// import sqlite3
	_ "github.com/mattn/go-sqlite3"
)

// table name
const (
	tableNameSignalEvents = "signal_events"
)

var DbConnection *sql.DB

func GetCandleTableName(productCode string, duration time.Duration) string {
	return fmt.Sprintf("%s_%s", productCode, duration)
}

func init() {
	var err error
	// open database using specified sql and database name
	DbConnection, err = sql.Open(config.Config.SQLDriver, config.Config.DbName)
	if err != nil {
		log.Fatalln(err)
	}
	// if database is not exist - create new table
	// DATETIME (when trade initiated) | product_code (BTC_USD) | side (buy/sell) | price (current price) | size (how much)
	cmd := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s (
            time DATETIME PRIMARY KEY NOT NULL,
            product_code STRING,
            side STRING,
            price FLOAT,
            size FLOAT)`, tableNameSignalEvents)
	_, err = DbConnection.Exec(cmd)
	if err != nil {
		log.Fatalln(err)
	}

	for _, duration := range config.Config.Durations {
		// BTC_USD_1m
		tableName := GetCandleTableName(config.Config.ProductCode, duration)
		c := fmt.Sprintf(`
            CREATE TABLE IF NOT EXISTS %s (
            time DATETIME PRIMARY KEY NOT NULL,
            open FLOAT,
            close FLOAT,
            high FLOAT,
            low FLOAT,
			volume FLOAT)`, tableName)
		_, err = DbConnection.Exec(c)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
