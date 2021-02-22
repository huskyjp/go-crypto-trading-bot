package controllers

import (
	"encoding/json"
	"fmt"
	"go-trading-bot/app/models"
	"go-trading-bot/config"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"text/template"
)

var templates = template.Must(template.ParseFiles("app/views/chart.html"))

func viewChartHandler(w http.ResponseWriter, r *http.Request) {

	limit := 100
	duration := "1s"
	durationTime := config.Config.Durations[duration]
	// get current Candle struct
	df, _ := models.GetAllCandle(config.Config.ProductCode, durationTime, limit)
	// insert df.Candles which contains current all candle info
	err := templates.ExecuteTemplate(w, "chart.html", df.Candles)
	if err != nil {
		log.Println("error happend")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// JSONError struct for manipulating live-candle-data
type JSONError struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// APIError creates error message when the connection failed
func APIError(w http.ResponseWriter, errMessage string, code int) {
	w.Header().Set("Content-Type", "application/json")
	// set error code such as 401
	w.WriteHeader(code)
	jsonError, err := json.Marshal(JSONError{Error: errMessage, Code: code})
	if err != nil {
		log.Fatal(err)
	}
	w.Write(jsonError)
}

// URL that shows candle information
var apiValidPath = regexp.MustCompile("^/api/candle/$")

// apiMakeHandler is a wrapper function that returns function or return error message if matched URL is ZERO
func apiMakeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := apiValidPath.FindStringSubmatch(r.URL.Path)
		if len(m) == 0 {
			APIError(w, "Not found", http.StatusNotFound)
		}
		// fn => apiCandleHandler
		fn(w, r)
	}
}

// apiCandleHandler creates
func apiCandleHandler(w http.ResponseWriter, r *http.Request) {
	// find product_code from browser
	productCode := r.URL.Query().Get("product_code")
	if productCode == "" {
		APIError(w, "No product_code param", http.StatusBadRequest)
		return
	}
	strLimit := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(strLimit)
	// limit max => 1000
	if strLimit == "" || err != nil || limit < 0 || limit > 1000 {
		limit = 1000
	}

	// set default duration as 1m
	duration := r.URL.Query().Get("duration")
	if duration == "" {
		duration = "1m"
	}
	// get durationTime from config file
	durationTime := config.Config.Durations[duration]
	// get candle struct with productCode, durationTime, and limit
	df, _ := models.GetAllCandle(productCode, durationTime, limit)

	// get sma query (i.e. get sma from URL)
	sma := r.URL.Query().Get("sma")
	// if it is exsisted
	if sma != "" {
		// set 3 periods
		strSmaPeriod1 := r.URL.Query().Get("smaPeriod1") // get period from client
		strSmaPeriod2 := r.URL.Query().Get("smaPeriod2")
		strSmaPeriod3 := r.URL.Query().Get("smaPeriod3")
		// convert into integer
		period1, err := strconv.Atoi(strSmaPeriod1)
		// default value: 7, 14, 50
		if strSmaPeriod1 == "" || err != nil || period1 < 0 {
			period1 = 7
		}
		period2, err := strconv.Atoi(strSmaPeriod2)
		if strSmaPeriod2 == "" || err != nil || period2 < 0 {
			period2 = 14
		}
		period3, err := strconv.Atoi(strSmaPeriod3)
		if strSmaPeriod3 == "" || err != nil || period3 < 0 {
			period3 = 50
		}
		// add each period to the AddSma struct
		df.AddSma(period1)
		df.AddSma(period2)
		df.AddSma(period3)
	}

	// get ema query
	ema := r.URL.Query().Get("ema")
	// if it exsists
	if ema != "" {
		strEmaPeriod1 := r.URL.Query().Get("emaPeriod1")
		strEmaPeriod2 := r.URL.Query().Get("emaPeriod2")
		strEmaPeriod3 := r.URL.Query().Get("emaPeriod3")
		period1, err := strconv.Atoi(strEmaPeriod1)
		if strEmaPeriod1 == "" || err != nil || period1 < 0 {
			period1 = 7
		}
		period2, err := strconv.Atoi(strEmaPeriod2)
		if strEmaPeriod2 == "" || err != nil || period2 < 0 {
			period2 = 14
		}
		period3, err := strconv.Atoi(strEmaPeriod3)
		if strEmaPeriod3 == "" || err != nil || period3 < 0 {
			period3 = 50
		}
		df.AddEma(period1)
		df.AddEma(period2)
		df.AddEma(period3)
	}

	// get bolinger bands query from client
	bbands := r.URL.Query().Get("bbands")
	// if it exists...
	if bbands != "" {
		strN := r.URL.Query().Get("bbandsN")
		strK := r.URL.Query().Get("bbandsK")
		n, err := strconv.Atoi(strN)
		if strN == "" || err != nil || n < 0 {
			n = 20
		}
		k, err := strconv.Atoi(strK)
		if strK == "" || err != nil || k < 0 {
			k = 2
		}
		df.AddBBands(n, float64(k))
	}

	// get ichimoku query from client
	ichimoku := r.URL.Query().Get("ichimoku")
	// if it exists...
	if ichimoku != "" {
		df.AddIchimoku()
	}

	rsi := r.URL.Query().Get("rsi")
	if rsi != "" {
		strPeriod := r.URL.Query().Get("rsiPeriod")
		period, err := strconv.Atoi(strPeriod)
		if strPeriod == "" || err != nil || period < 0 {
			period = 14
		}
		df.AddRsi(period)
	}

	macd := r.URL.Query().Get("macd")
	if macd != "" {
		strPeriod1 := r.URL.Query().Get("macdPeriod1")
		strPeriod2 := r.URL.Query().Get("macdPeriod2")
		strPeriod3 := r.URL.Query().Get("macdPeriod3")
		period1, err := strconv.Atoi(strPeriod1)
		if strPeriod1 == "" || err != nil || period1 < 0 {
			period1 = 12
		}
		period2, err := strconv.Atoi(strPeriod2)
		if strPeriod2 == "" || err != nil || period2 < 0 {
			period2 = 26
		}
		period3, err := strconv.Atoi(strPeriod3)
		if strPeriod3 == "" || err != nil || period3 < 0 {
			period3 = 9
		}
		df.AddMacd(period1, period2, period3)
	}

	hv := r.URL.Query().Get("hv")
	if hv != "" {
		strPeriod1 := r.URL.Query().Get("hvPeriod1")
		strPeriod2 := r.URL.Query().Get("hvPeriod2")
		strPeriod3 := r.URL.Query().Get("hvPeriod3")
		period1, err := strconv.Atoi(strPeriod1)
		if strPeriod1 == "" || err != nil || period1 < 0 {
			period1 = 21
		}
		period2, err := strconv.Atoi(strPeriod2)
		if strPeriod2 == "" || err != nil || period2 < 0 {
			period2 = 63
		}
		period3, err := strconv.Atoi(strPeriod3)
		if strPeriod3 == "" || err != nil || period3 < 0 {
			period3 = 252
		}
		df.AddHv(period1)
		df.AddHv(period2)
		df.AddHv(period3)
	}

	events := r.URL.Query().Get("events")
	if events != "" {
		if config.Config.BackTest {
			df.Events = Ai.SignalEvents.GetAfter(df.Candles[0].Time)
			// when we have profit with the algorithm
			// if performance > 0 {
			// 	df.Events = df.BackTestBb(p1, p2)
			// }
		} else {
			firstTime := df.Candles[0].Time
			df.AddEvents(firstTime)
		}
	}

	// Convert Candle struct to JSON
	candleJSON, err := json.Marshal(df)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(candleJSON)
}

// StartWebServer initiate the chart UI
func StartWebServer() error {
	http.HandleFunc("/api/candle/", apiMakeHandler(apiCandleHandler))
	http.HandleFunc("/chart/", viewChartHandler)
	return http.ListenAndServe(fmt.Sprintf(":%d", config.Config.Port), nil)
}
