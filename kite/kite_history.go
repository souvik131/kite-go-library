package kite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/souvik131/kite-go-library/requests"
)

func (kite *Kite) GetHistoricalMinutelyData(ctx *context.Context, token uint32, interval string, startDate string, endDate string) ([]*Candle, error) {

	k := *(*kite).Creds

	url := fmt.Sprintf("%v/instruments/historical/%v/minute?from=%v&to=%v&oi=1", k["Url"], token, startDate, endDate)

	headers := map[string]string{
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept-Encoding": "gzip, deflate",
		"Host":            "kite.zerodha.com",
		"Accept":          "*/*",
	}
	headers["authorization"] = k["Token"]
	headers["content-type"] = "application/x-www-form-urlencoded"

	res, code, cookie, err := requests.GetWithCookies(ctx, url, headers, k["Cookie"])
	k["Cookie"] = cookie
	if err != nil {
		return nil, err
	}

	var respData *CandlesResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return nil, err
	}

	if code == 200 && respData.Data != nil {
		candles := []*Candle{}
		for _, candle := range respData.Data.Candles {
			c := &Candle{}
			for i, d := range *candle {
				switch i {
				case 0:
					layout := "2006-01-02T15:04:05-0700"

					t, err := time.Parse(layout, fmt.Sprintf("%v", *d))
					if err != nil {
						return nil, err
					}
					c.Timestamp = t.UnixNano()
				case 1:
					c.Open, err = strconv.ParseFloat(fmt.Sprintf("%v", *d), 64)
					if err != nil {
						return nil, err
					}
				case 2:
					c.High, err = strconv.ParseFloat(fmt.Sprintf("%v", *d), 64)
					if err != nil {
						return nil, err
					}
				case 3:
					c.Low, err = strconv.ParseFloat(fmt.Sprintf("%v", *d), 64)
					if err != nil {
						return nil, err
					}
				case 4:
					c.Close, err = strconv.ParseFloat(fmt.Sprintf("%v", *d), 64)
					if err != nil {
						return nil, err
					}
				case 5:
					// Parse Volume as float64 first to handle scientific notation
					volumeFloat, err := strconv.ParseFloat(fmt.Sprintf("%v", *d), 64)
					if err != nil {
						return nil, err
					}
					c.Volume = uint64(volumeFloat)
				case 6:
					// Parse OI as float64 first to handle scientific notation
					oiFloat, err := strconv.ParseFloat(fmt.Sprintf("%v", *d), 64)
					if err != nil {
						return nil, err
					}
					c.OI = uint64(oiFloat)
				}

				candles = append(candles, c)
			}
		}

		return candles, nil
	}
	return nil, errors.New(respData.Status + ":" + respData.Message)
}

// GetHistoricalData - Enhanced function that accepts exchange and trading symbol
func (kite *Kite) GetHistoricalData(ctx *context.Context, exchange string, tradingSymbol string, interval string, startDate string, endDate string) ([]*Candle, error) {
	// Find the instrument token for the given symbol
	if BrokerInstrumentTokens == nil {
		return nil, fmt.Errorf("instrument tokens not loaded")
	}

	symbolKey := exchange + ":" + tradingSymbol
	instrument, exists := (*BrokerInstrumentTokens)[symbolKey]
	if !exists {
		return nil, fmt.Errorf("instrument %s not found", symbolKey)
	}

	// Use the existing function with the found token
	return kite.GetHistoricalMinutelyData(ctx, instrument.Token, interval, startDate, endDate)
}
