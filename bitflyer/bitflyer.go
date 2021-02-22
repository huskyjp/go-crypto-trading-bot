package bitflyer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const baseURL = "https://api.bitflyer.com/v1/"

// create struct which is like a object
type APIClient struct {
	key        string
	secret     string
	httpClient *http.Client
}

// Constractor: pass apikey and secreat as string, the nreturn pointer to APIClient
func New(key, secret string) *APIClient {
	apiClient := &APIClient{key, secret, &http.Client{}}
	return apiClient
}

// takes APIClient struct and method, endpoint and boty as byte, returns map of string/string as HEADER
func (api APIClient) header(method, endpoint string, body []byte) map[string]string {
	// create timestamp as string
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	log.Println(timestamp)
	message := timestamp + method + endpoint + string(body)

	// create crypted sha256 utilizing user's secret
	mac := hmac.New(sha256.New, []byte(api.secret))
	// add timestamp etc... to encrypted object to hash value
	mac.Write([]byte(message))
	// add nil to mac and hex.Enconde => convert to string with 16
	sign := hex.EncodeToString(mac.Sum(nil))
	return map[string]string{
		"ACCESS-KEY":       api.key,
		"ACCESS-TIMESTAMP": timestamp,
		"ACCESS-SIGN":      sign,
		"Content-Type":     "application/json",
	}
}

// Use HEADER and Create Request
// parameter example: "GET", /me/deposit, map[string]string{}, nil
func (api *APIClient) doRequest(method, urlPath string, query map[string]string, data []byte) (body []byte, err error) {
	// check if baseurl is not nil
	baseURL, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	// check api url is not nil
	apiURL, err := url.Parse(urlPath)
	if err != nil {
		return
	}
	// create request url
	endpoint := baseURL.ResolveReference(apiURL).String()
	//log.Printf("action=doRequest endpoint=%s", endpoint)

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(data))
	if err != nil {
		return
	}

	// apiURLにあるMAP化されているデータをチェック
	// reference, _ := url.Parse("/test?a=1&b=2")、今回はまだ何もないので、渡されてくるqueryを入れ込む
	// q にはmap型が宣言される

	// reqのURL内にある、独立したクエリの部分のデータを受け取る。
	// 今回は入ってないから、受け取ったマップ型のをRangeにかけて、入れ込む

	q := req.URL.Query()
	for key, value := range query {
		q.Add(key, value)
	}
	// reqに対してデータを戻す。ただ、＆とかが入ってると上手く入らないのでEncodesしてから戻す必要性がある
	req.URL.RawQuery = q.Encode()

	for key, value := range api.header(method, req.URL.RequestURI(), data) {
		req.Header.Add(key, value)
	}

	// StructからClientを呼び出して、今まで作ってきたデータを渡してリクエストをする
	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	// if no error - close the resp
	defer resp.Body.Close()
	// read the body of response
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type Balance struct {
	CurrentCode string  `json:"currency_code"`
	Amount      float64 `json:"amount"`
	Available   float64 `json:"available"`
}

func (api *APIClient) GetBalance() ([]Balance, error) {
	url := "me/getbalance"
	// call function: func (api...) doRequest...
	resp, err := api.doRequest("GET", url, map[string]string{}, nil)
	log.Printf("url=%s resp=%s", url, string(resp))
	if err != nil {
		log.Printf("action=GetBalance err=%s", err.Error())
		return nil, err
	}
	var balance []Balance
	// get json object from response and convert it into Balance struct
	err = json.Unmarshal(resp, &balance)
	if err != nil {
		log.Printf("action=GetBalance err=%s", err.Error())
		return nil, err
	}
	return balance, nil
}

// get ticker information - from json-to-go
type Ticker struct {
	ProductCode     string  `json:"product_code"`
	State           string  `json:"state"`
	Timestamp       string  `json:"timestamp"`
	TickID          int     `json:"tick_id"`
	BestBid         float64 `json:"best_bid"`
	BestAsk         float64 `json:"best_ask"`
	BestBidSize     float64 `json:"best_bid_size"`
	BestAskSize     float64 `json:"best_ask_size"`
	TotalBidDepth   float64 `json:"total_bid_depth"`
	TotalAskDepth   float64 `json:"total_ask_depth"`
	MarketBidSize   float64 `json:"market_bid_size"`
	MarketAskSize   float64 `json:"market_ask_size"`
	Ltp             float64 `json:"ltp"`
	Volume          float64 `json:"volume"`
	VolumeByProduct float64 `json:"volume_by_product"`
}

func (t *Ticker) GetMidPrice() float64 {
	return (float64(t.BestBid) + float64(t.BestAsk)) / 2
}

func (t *Ticker) DateTime() time.Time {
	dateTime, err := time.Parse(time.RFC3339, t.Timestamp)
	if err != nil {
		log.Printf("action=DateTime, err=%s", err.Error())
	}
	return dateTime
}

func (t *Ticker) TruncateDateTime(duration time.Duration) time.Time {
	return t.DateTime().Truncate(duration)
}

// takes product code and return Ticker value
func (api *APIClient) GetTicker(productCode string) (*Ticker, error) {
	url := "ticker"
	// call function: func (api...) doRequest...
	resp, err := api.doRequest("GET", url, map[string]string{"product_code": productCode}, nil)
	if err != nil {
		return nil, err
	}
	var ticker Ticker
	err = json.Unmarshal(resp, &ticker)
	if err != nil {
		return nil, err
	}
	return &ticker, nil
}

type JsonRPC2 struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Result  interface{} `json:"result,omitempty"`
	Id      *int        `json:"id,omitempty"`
}

type SubscribeParams struct {
	Channel string `json:"channel"`
}

func (api *APIClient) GetRealTimeTicker(symbol string, ch chan<- Ticker) {
	u := url.URL{Scheme: "wss", Host: "ws.lightstream.bitflyer.com", Path: "/json-rpc"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	channel := fmt.Sprintf("lightning_ticker_%s", symbol)
	if err := c.WriteJSON(&JsonRPC2{Version: "2.0", Method: "subscribe", Params: &SubscribeParams{channel}}); err != nil {
		log.Fatal("subscribe:", err)
		return
	}

OUTER:
	for {
		message := new(JsonRPC2)
		if err := c.ReadJSON(message); err != nil {
			log.Println("read:", err)
			return
		}

		if message.Method == "channelMessage" {
			switch v := message.Params.(type) {
			case map[string]interface{}:
				for key, binary := range v {
					if key == "message" {
						marshaTic, err := json.Marshal(binary)
						if err != nil {
							continue OUTER
						}
						var ticker Ticker
						if err := json.Unmarshal(marshaTic, &ticker); err != nil {
							continue OUTER
						}
						ch <- ticker
					}
				}
			}
		}
	}
}

// Order struct for creating order for trading
type Order struct {
	ID                     int     `json:"id"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
	ProductCode            string  `json:"product_code"`
	ChildOrderType         string  `json:"child_order_type"`
	Side                   string  `json:"side"`
	Price                  float64 `json:"price"`
	Size                   float64 `json:"size"`
	MinuteToExpires        int     `json:"minute_to_expire"`
	TimeInForce            string  `json:"time_in_force"`
	Status                 string  `json:"status"`
	ErrorMessage           string  `json:"error_message"`
	AveragePrice           float64 `json:"average_price"`
	ChildOrderState        string  `json:"child_order_state"`
	ExpireDate             string  `json:"expire_date"`
	ChildOrderDate         string  `json:"child_order_date"`
	OutstandingSize        float64 `json:"outstanding_size"`
	CancelSize             float64 `json:"cancel_size"`
	ExecutedSize           float64 `json:"executed_size"`
	TotalCommission        float64 `json:"total_commission"`
	Count                  int     `json:"count"`
	Before                 int     `json:"before"`
	After                  int     `json:"after"`
}

// response when we order
type ResponseSendChildOrder struct {
	ChildOrderAcceptanceID string `json:"child_order_acceptance_id`
}

// create order!
func (api *APIClient) SendOrder(order *Order) (*ResponseSendChildOrder, error) {
	// 入ってくるオーダーをＪＳＯＮにする
	data, _ := json.Marshal(order)
	url := "me/sendchildorder"
	resp, _ := api.doRequest("POST", url, map[string]string{}, data)

	var response ResponseSendChildOrder
	_ = json.Unmarshal(resp, &response)

	return &response, nil
}

// order list
func (api *APIClient) ListOrder(query map[string]string) ([]Order, error) {
	resp, err := api.doRequest("GET", "me/getchildorders", query, nil)
	if err != nil {
		return nil, err
	}

	var responseListOrder []Order
	err = json.Unmarshal(resp, &responseListOrder)
	if err != nil {
		return nil, err
	}
	return responseListOrder, nil
}
