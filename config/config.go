package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/ini.v1"
)

// ConfigList struct read initial configuration which contains APIKEY and APISECRET
// and assign those into GO struct as configKeyAndSecret
type ConfigList struct {
	ApiKey      string
	ApiSecret   string
	LogFile     string
	ProductCode string

	TradeDuration time.Duration            // manually select trade duration
	Durations     map[string]time.Duration // map of duration choices
	DbName        string
	SQLDriver     string
	Port          int

	BackTest         bool
	UsePercent       float64
	DataLimit        int
	StopLimitPercent float64
	NumRanking       int
}

// Config to access to the struct list of configuration list
var Config ConfigList

// Initialize the ConfigList struct
func init() {
	cfg, err := ini.Load("config.ini") // load config.ini file to get the defined value
	if err != nil {
		log.Printf("Failed to read file, something is wrong: %v", err)
		os.Exit(1)
	}
	// define each durations
	durations := map[string]time.Duration{
		"1s":  time.Second,
		"1m":  time.Minute,
		"2m":  time.Minute * 2,
		"5m":  time.Minute * 5,
		"15m": time.Minute * 15,
		"30m": time.Minute * 30,
		"1h":  time.Hour,
	}

	Config = ConfigList{
		ApiKey:           cfg.Section("bitflyer").Key("api_key").String(),
		ApiSecret:        cfg.Section("bitflyer").Key("api_secret").String(),
		LogFile:          cfg.Section("gotradingbot").Key("log_file").String(),
		ProductCode:      cfg.Section("gotradingbot").Key("product_code").String(),
		Durations:        durations,
		TradeDuration:    durations[cfg.Section("gotradingbot").Key("trade_duration").String()],
		DbName:           cfg.Section("db").Key("name").String(),
		SQLDriver:        cfg.Section("db").Key("driver").String(),
		Port:             cfg.Section("web").Key("port").MustInt(),
		BackTest:         cfg.Section("gotradingbot").Key("back_test").MustBool(),
		UsePercent:       cfg.Section("gotradingbot").Key("use_percent").MustFloat64(),
		DataLimit:        cfg.Section("gotradingbot").Key("data_limit").MustInt(),
		StopLimitPercent: cfg.Section("gotradingbot").Key("stop_limit_percent").MustFloat64(),
		NumRanking:       cfg.Section("gotradingbot").Key("num_ranking").MustInt(),
	}
}
