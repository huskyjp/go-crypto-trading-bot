# Go Crypto Trading Bot
Simple cryptocurrency trading bot.
Algorithm includes: `SMA, EMA, Bollinger Bands, RSI, Historical Volatirity`.
*Highly reccomend to customize the algorithms, set `backtest = true` for backtesting before trading.

## Documentation

- [Install Golang](https://golang.org/doc/install)
- [Install SQLite](https://www.sqlite.org/index.html)
- [bitFlyter API Documentation](https://lightning.bitflyer.com/docs?lang=en)

# Get Started
Insert own `api_key` and `api_secret` at `config.ini` file.
Customize algorithms as you wish at `ai.go`.

## Run with Golang
Run `go run main.go`.

# API
- Endpoint `https://api.bitflyer.com/v1/`

### Private API
|      Support       | Method |     Endpoint                 |
| ------------------ | ------ | -----------------            |
| :white_check_mark: | GET    | /v1/me/getbalance            |
| :white_check_mark: | GET    | /v1/me/sendchildorder        |

### Public API
|      Support       | Method |     Endpoint                 |
| ------------------ | ------ | -----------------            |
| :white_check_mark: | GET    | /v1/ticker                   |


### JSON-RPC 2.0 over WebSocket
|      Support       | Method (Client/Server)    |     Endpoint                                               |
| ------------------ | ------                    | -----------------                                          |
| :white_check_mark: | subscribe/channelMessage  | wss://ws.lightstream.bitflyer.com/json-rpc                 |


# Default Algorithm

- [Simple Moving Average (SMA)](https://www.investopedia.com/terms/s/sma.asp)
- [Exponential Moving Average (EMA)](https://www.investopedia.com/terms/e/ema.asp)
- [Bollinger Bands](https://www.investopedia.com/terms/b/bollingerbands.asp)
- [Ichimoku Cloud](https://www.investopedia.com/terms/i/ichimoku-cloud.asp)
- [Volume](https://www.investopedia.com/terms/v/volume.asp)
- [Relative Strength Index (RSI)](https://www.investopedia.com/terms/r/rsi.asp)
- [Historical Volatility (HV)](https://www.investopedia.com/terms/h/historicalvolatility.asp)


# Note
- talib
- websocket
- json-rpc2.0
- REST API
- Semaphore
- hmac
- SQLite3
- AI Model (Auto Trade Algorithm)
- Google Chart

# Disclaimer
We do not accept any responsibility or liability for the trading loss, please use the program at your own risk.