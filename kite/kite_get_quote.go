package kite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/souvik131/kite-go-library/requests"
)

func (kite *Kite) GetQuote(ctx *context.Context, exchange string, tradingSymbol string) (*Quote, error) {

	k := *(*kite).Creds

	// For WEB login type, use WebSocket pipeline data with prioritization
	if k["LoginType"] == "WEB" {
		return kite.getQuoteFromWebSocketWithPriority(ctx, exchange, tradingSymbol)
	}

	// For API login type, try WebSocket first, fallback to API
	quote, err := kite.getQuoteFromWebSocketWithPriority(ctx, exchange, tradingSymbol)
	if err == nil {
		return quote, nil
	}

	// Fallback to API call for non-WEB login types
	url := k["Url"] + "/quote?i=" + exchange + ":" + url.QueryEscape(tradingSymbol)
	headers := make(map[string]string)
	headers["authorization"] = k["Token"]
	headers["content-type"] = "application/x-www-form-urlencoded"

	response, _, err := requests.Get(ctx, url, headers)

	if err != nil {
		return nil, err
	}
	var respData *QuoteResponsePayload
	err = json.Unmarshal(response, &respData)
	if err != nil {
		return nil, err
	}

	if respData.Data == nil {
		return nil, errors.New(respData.Message)

	}

	return (respData.Data)[exchange+":"+tradingSymbol], nil
}

func (kite *Kite) GetLastPrice(ctx *context.Context, exchange string, tradingSymbol string) (float64, error) {

	k := *(*kite).Creds

	// For WEB login type, use WebSocket pipeline data with prioritization
	if k["LoginType"] == "WEB" {
		return kite.getLastPriceFromWebSocketWithPriority(ctx, exchange, tradingSymbol)
	}

	// For API login type, try WebSocket first, fallback to API
	price, err := kite.getLastPriceFromWebSocketWithPriority(ctx, exchange, tradingSymbol)
	if err == nil {
		return price, nil
	}

	// Fallback to API call for non-WEB login types
	url := k["Url"] + "/quote?i=" + exchange + ":" + url.QueryEscape(tradingSymbol)
	headers := make(map[string]string)
	headers["authorization"] = k["Token"]
	headers["content-type"] = "application/x-www-form-urlencoded"

	response, _, err := requests.Get(ctx, url, headers)

	if err != nil {
		return 0.0, err
	}
	var respData *QuoteResponsePayload
	err = json.Unmarshal(response, &respData)
	if err != nil {
		return 0, err
	}

	if respData.Data == nil {
		return 0, errors.New(respData.Message)

	}
	price = (respData.Data)[exchange+":"+tradingSymbol].LastPrice

	return price, nil
}

// getQuoteFromWebSocket retrieves quote data from the WebSocket pipeline
func (kite *Kite) getQuoteFromWebSocket(exchange string, tradingSymbol string) (*Quote, error) {
	if kite.TickSymbolMap == nil {
		return nil, fmt.Errorf("websocket data not available")
	}

	// Use mutex to safely read from TickSymbolMap
	kite.TickSymbolMapMutex.RLock()
	defer kite.TickSymbolMapMutex.RUnlock()

	// Try to find the ticker data in the WebSocket feed
	symbolKey := exchange + ":" + tradingSymbol
	ticker, exists := kite.TickSymbolMap[symbolKey]
	if !exists {
		// Also try with just the trading symbol
		ticker, exists = kite.TickSymbolMap[tradingSymbol]
		if !exists {
			return nil, fmt.Errorf("symbol %s not found in websocket feed", symbolKey)
		}
	}

	// Convert KiteTicker to Quote format
	quote := &Quote{
		LastPrice: ticker.LastPrice,
		Depth: struct {
			Buy []struct {
				Price float64 `json:"price"`
			} `json:"buy"`
			Sell []struct {
				Price float64 `json:"price"`
			} `json:"sell"`
		}{},
	}

	// Convert depth data
	for _, buyOrder := range ticker.Depth.Buy {
		quote.Depth.Buy = append(quote.Depth.Buy, struct {
			Price float64 `json:"price"`
		}{Price: buyOrder.Price})
	}

	for _, sellOrder := range ticker.Depth.Sell {
		quote.Depth.Sell = append(quote.Depth.Sell, struct {
			Price float64 `json:"price"`
		}{Price: sellOrder.Price})
	}

	return quote, nil
}

// getLastPriceFromWebSocket retrieves last price from the WebSocket pipeline
func (kite *Kite) getLastPriceFromWebSocket(exchange string, tradingSymbol string) (float64, error) {
	if kite.TickSymbolMap == nil {
		return 0.0, fmt.Errorf("websocket data not available")
	}

	// Use mutex to safely read from TickSymbolMap
	kite.TickSymbolMapMutex.RLock()
	defer kite.TickSymbolMapMutex.RUnlock()

	// Try to find the ticker data in the WebSocket feed
	symbolKey := exchange + ":" + tradingSymbol
	ticker, exists := kite.TickSymbolMap[symbolKey]
	if !exists {
		// Also try with just the trading symbol
		ticker, exists = kite.TickSymbolMap[tradingSymbol]
		if !exists {
			return 0.0, fmt.Errorf("symbol %s not found in websocket feed", symbolKey)
		}
	}

	return ticker.LastPrice, nil
}

// RequestQuoteFromWebSocket requests a specific instrument to be added to the WebSocket feed
func (kite *Kite) RequestQuoteFromWebSocket(ctx *context.Context, exchange string, tradingSymbol string) error {
	// Find the instrument token for the given symbol
	if BrokerInstrumentTokens == nil {
		return fmt.Errorf("instrument tokens not loaded")
	}

	symbolKey := exchange + ":" + tradingSymbol
	instrument, exists := (*BrokerInstrumentTokens)[symbolKey]
	if !exists {
		return fmt.Errorf("instrument %s not found", symbolKey)
	}

	// If we have active ticker clients, subscribe to this token
	if len(kite.TickerClients) > 0 {
		for _, client := range kite.TickerClients {
			if client != nil {
				// Subscribe to quote data for this token
				err := client.SubscribeQuote(ctx, []uint32{instrument.Token})
				if err != nil {
					return fmt.Errorf("failed to subscribe to token %d: %v", instrument.Token, err)
				}
			}
		}
	}

	// Wait a short time for data to arrive
	time.Sleep(100 * time.Millisecond)

	return nil
}

// GetQuoteWithSubscription attempts to get quote data and subscribes if not available
func (kite *Kite) GetQuoteWithSubscription(ctx *context.Context, exchange string, tradingSymbol string) (*Quote, error) {
	// First try to get from existing data
	quote, err := kite.getQuoteFromWebSocket(exchange, tradingSymbol)
	if err == nil {
		return quote, nil
	}

	// If not available, request subscription and try again
	err = kite.RequestQuoteFromWebSocket(ctx, exchange, tradingSymbol)
	if err != nil {
		// If subscription fails, fallback to original method
		return kite.GetQuote(ctx, exchange, tradingSymbol)
	}

	// Try again after subscription
	quote, err = kite.getQuoteFromWebSocket(exchange, tradingSymbol)
	if err != nil {
		// If still not available, fallback to original method
		return kite.GetQuote(ctx, exchange, tradingSymbol)
	}

	return quote, nil
}

// GetLastPriceWithSubscription attempts to get last price and subscribes if not available
func (kite *Kite) GetLastPriceWithSubscription(ctx *context.Context, exchange string, tradingSymbol string) (float64, error) {
	// First try to get from existing data
	price, err := kite.getLastPriceFromWebSocket(exchange, tradingSymbol)
	if err == nil {
		return price, nil
	}

	// If not available, request subscription and try again
	err = kite.RequestQuoteFromWebSocket(ctx, exchange, tradingSymbol)
	if err != nil {
		// If subscription fails, fallback to original method
		return kite.GetLastPrice(ctx, exchange, tradingSymbol)
	}

	// Try again after subscription
	price, err = kite.getLastPriceFromWebSocket(exchange, tradingSymbol)
	if err != nil {
		// If still not available, fallback to original method
		return kite.GetLastPrice(ctx, exchange, tradingSymbol)
	}

	return price, nil
}

// getQuoteFromWebSocketWithPriority retrieves quote data by adding to batch and waiting
func (kite *Kite) getQuoteFromWebSocketWithPriority(ctx *context.Context, exchange string, tradingSymbol string) (*Quote, error) {
	// Initialize WebSocket if not available
	if kite.TickSymbolMap == nil {
		return nil, fmt.Errorf("websocket not initialized yet")
	}

	// Try immediate lookup first
	quote, err := kite.getQuoteFromWebSocket(exchange, tradingSymbol)
	if err == nil {
		return quote, nil
	}

	// If not found, add to current batch and wait for data
	err = kite.addToBatchAndWait(ctx, exchange, tradingSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to add symbol to batch: %v", err)
	}

	// Try again after adding to batch
	quote, err = kite.getQuoteFromWebSocket(exchange, tradingSymbol)
	if err != nil {
		return nil, fmt.Errorf("symbol %s still not available after adding to batch", exchange+":"+tradingSymbol)
	}

	return quote, nil
}

// getLastPriceFromWebSocketWithPriority retrieves last price by adding to batch and waiting
func (kite *Kite) getLastPriceFromWebSocketWithPriority(ctx *context.Context, exchange string, tradingSymbol string) (float64, error) {
	// Initialize WebSocket if not available
	if kite.TickSymbolMap == nil {
		return 0.0, fmt.Errorf("websocket not initialized yet")
	}

	// Try immediate lookup first
	price, err := kite.getLastPriceFromWebSocket(exchange, tradingSymbol)
	if err == nil {
		return price, nil
	}

	// If not found, add to current batch and wait for data
	err = kite.addToBatchAndWait(ctx, exchange, tradingSymbol)
	if err != nil {
		return 0.0, fmt.Errorf("failed to add symbol to batch: %v", err)
	}

	// Try again after adding to batch
	price, err = kite.getLastPriceFromWebSocket(exchange, tradingSymbol)
	if err != nil {
		return 0.0, fmt.Errorf("symbol %s still not available after adding to batch", exchange+":"+tradingSymbol)
	}

	return price, nil
}

// addToBatchAndWait adds the symbol to the current WebSocket batch and waits for data
func (kite *Kite) addToBatchAndWait(ctx *context.Context, exchange string, tradingSymbol string) error {
	// Find the instrument token for the given symbol
	if BrokerInstrumentTokens == nil {
		return fmt.Errorf("instrument tokens not loaded")
	}

	symbolKey := exchange + ":" + tradingSymbol
	instrument, exists := (*BrokerInstrumentTokens)[symbolKey]
	if !exists {
		return fmt.Errorf("instrument %s not found", symbolKey)
	}

	// Check if we have active ticker clients
	if len(kite.TickerClients) == 0 {
		return fmt.Errorf("no active ticker clients available")
	}

	// Add this token to the current batch by subscribing to it
	for _, client := range kite.TickerClients {
		if client != nil {
			// Subscribe to full data for this token
			err := client.SubscribeFull(ctx, []uint32{instrument.Token})
			if err != nil {
				return fmt.Errorf("failed to subscribe to token %d: %v", instrument.Token, err)
			}
		}
	}

	// Wait for data to appear in the map
	maxWaitTime := 5 * time.Second
	checkInterval := 100 * time.Millisecond
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		// Check if data is now available using mutex
		kite.TickSymbolMapMutex.RLock()
		ticker, exists := kite.TickSymbolMap[symbolKey]
		if exists && ticker.LastPrice > 0 {
			kite.TickSymbolMapMutex.RUnlock()
			return nil
		}
		// Also check with just trading symbol
		ticker, exists = kite.TickSymbolMap[tradingSymbol]
		if exists && ticker.LastPrice > 0 {
			kite.TickSymbolMapMutex.RUnlock()
			return nil
		}
		kite.TickSymbolMapMutex.RUnlock()

		// Wait before next check
		time.Sleep(checkInterval)
	}

	return fmt.Errorf("timeout waiting for symbol %s data after adding to batch", symbolKey)
}
