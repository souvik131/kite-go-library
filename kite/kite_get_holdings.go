package kite

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/souvik131/kite-go-library/requests"
)

type Holding struct {
	TradingSymbol      string  `json:"tradingsymbol"`
	Exchange           string  `json:"exchange"`
	InstrumentToken    uint32  `json:"instrument_token"`
	ISIN               string  `json:"isin"`
	Product            string  `json:"product"`
	Price              float64 `json:"price"`
	Quantity           int64   `json:"quantity"`
	UsedQuantity       int64   `json:"used_quantity"`
	T1Quantity         int64   `json:"t1_quantity"`
	RealisedQuantity   float64 `json:"realised_quantity"`
	OpeningQuantity    int64   `json:"opening_quantity"`
	ShortQuantity      int64   `json:"short_quantity"`
	CollateralQuantity int64   `json:"collateral_quantity"`
	CollateralType     string  `json:"collateral_type"`
	Discrepancy        bool    `json:"discrepancy"`
	AveragePrice       float64 `json:"average_price"`
	LastPrice          float64 `json:"last_price"`
	ClosePrice         float64 `json:"close_price"`
	PnL                float64 `json:"pnl"`
	DayChange          float64 `json:"day_change"`
	DayChangePercent   float64 `json:"day_change_percentage"`
}

type HoldingsResponsePayload struct {
	Status    string     `json:"error"`
	Message   string     `json:"message"`
	ErrorType string     `json:"error_type"`
	Data      []*Holding `json:"data"`
}

func (kite *Kite) GetHoldings(ctx *context.Context) ([]*Holding, error) {
	k := *(*kite).Creds
	url := k["Url"] + "/portfolio/holdings"

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

	var respData *HoldingsResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return nil, err
	}

	if code == 200 && respData.Data != nil {
		return respData.Data, nil
	}
	return nil, errors.New(respData.Status + ":" + respData.Message)
}
