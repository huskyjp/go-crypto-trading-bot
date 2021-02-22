package main

import (
	"fmt"
	"go-trading-bot/app/controllers"
	"go-trading-bot/app/models"
	"go-trading-bot/config"
	"go-trading-bot/utils"
	"log"
	"time"
)

func main() {
	df, _ := models.GetAllCandle(config.Config.ProductCode, time.Minute, 365)
	fmt.Printf("%+v\n", df.OptimizeParams())
	utils.LoggingSettings(config.Config.LogFile)
	controllers.StreamIngestionData()
	log.Println(controllers.StartWebServer())
}
