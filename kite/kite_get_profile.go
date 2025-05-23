package kite

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/souvik131/kite-go-library/requests"
)

type Profile struct {
	UserID        string   `json:"user_id"`
	UserType      string   `json:"user_type"`
	Email         string   `json:"email"`
	UserName      string   `json:"user_name"`
	UserShortname string   `json:"user_shortname"`
	Broker        string   `json:"broker"`
	Exchanges     []string `json:"exchanges"`
	Products      []string `json:"products"`
	OrderTypes    []string `json:"order_types"`
	Avatar        string   `json:"avatar"`
	Meta          struct {
		DematConsent string `json:"demat_consent"`
	} `json:"meta"`
}

type ProfileResponsePayload struct {
	Status    string   `json:"error"`
	Message   string   `json:"message"`
	ErrorType string   `json:"error_type"`
	Data      *Profile `json:"data"`
}

func (kite *Kite) GetProfile(ctx *context.Context) (*Profile, error) {
	k := *(*kite).Creds
	url := k["Url"] + "/user/profile"

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

	var respData *ProfileResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return nil, err
	}

	if code == 200 && respData.Data != nil {
		return respData.Data, nil
	}
	return nil, errors.New(respData.Status + ":" + respData.Message)
}
