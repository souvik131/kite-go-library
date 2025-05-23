package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/souvik131/kite-go-library/engine"
	"github.com/souvik131/kite-go-library/kite"
)

var kiteClient *kite.Kite = &kite.Kite{}

func main() {
	// Load environment variables
	if os.Getenv("TA_ID") == "" {
		godotenv.Load()
	}
	log.Println("Login type:", os.Getenv("TA_LOGINTYPE"))

	// Create MCP server
	srv := server.NewMCPServer("kite-server", "1.0.0")

	// Register all Kite capabilities as MCP tools
	ctx := context.Background()

	err := kiteClient.Login(&ctx)
	if err != nil {
		log.Print(err)
		return
	}
	go engine.Write(&ctx, kiteClient)
	<-time.After(time.Second * 5)
	registerKiteTools(&ctx, srv)

	// Start the MCP server via stdio
	log.Println("Starting Kite MCP Server...")
	if err := server.ServeStdio(srv); err != nil {
		log.Fatalf("Failed to start MCP server: %v", err)
	}
}

func registerKiteTools(ctx *context.Context, srv *server.MCPServer) {
	// Get user ID for tool naming
	userID := os.Getenv("TA_ID")
	if userID == "" {
		userID = "default"
	}

	// Get Margin tool
	marginTool := mcp.NewTool(fmt.Sprintf("kite_get_margin_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get account margin details for user %s", userID)),
	)
	srv.AddTool(marginTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		margin, err := kiteClient.GetMargin(&ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get margin: %v", err)), nil
		}

		resultBytes, _ := json.Marshal(margin)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Positions tool
	positionsTool := mcp.NewTool(fmt.Sprintf("kite_get_positions_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get current positions for user %s", userID)),
	)
	srv.AddTool(positionsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		err := kiteClient.GetPositions(&ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get positions: %v", err)), nil
		}

		resultBytes, _ := json.Marshal(kiteClient.Positions)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Orders tool
	ordersTool := mcp.NewTool(fmt.Sprintf("kite_get_orders_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get all orders for user %s", userID)),
	)
	srv.AddTool(ordersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orders, err := kiteClient.GetOrders(&ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get orders: %v", err)), nil
		}

		resultBytes, _ := json.Marshal(orders)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Order History tool
	orderHistoryTool := mcp.NewTool(fmt.Sprintf("kite_get_order_history_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get order history for a specific order for user %s", userID)),
		mcp.WithString("order_id", mcp.Description("The order ID to get history for"), mcp.Required()),
	)
	srv.AddTool(orderHistoryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orderID, err := request.RequireString("order_id")
		if err != nil {
			return mcp.NewToolResultError("order_id is required"), nil
		}

		history, err := kiteClient.GetOrderHistory(&ctx, orderID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get order history: %v", err)), nil
		}

		resultBytes, _ := json.Marshal(history)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Quote tool
	quoteTool := mcp.NewTool(fmt.Sprintf("kite_get_quote_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get quote for a trading symbol for user %s", userID)),
		mcp.WithString("exchange", mcp.Description("Exchange (e.g., NSE, BSE, NFO, BFO)"), mcp.Required()),
		mcp.WithString("trading_symbol", mcp.Description("Trading symbol (e.g., RELIANCE, NIFTY24DEC24000CE)"), mcp.Required()),
	)
	srv.AddTool(quoteTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		exchange, err := request.RequireString("exchange")
		if err != nil {
			return mcp.NewToolResultError("exchange is required"), nil
		}

		tradingSymbol, err := request.RequireString("trading_symbol")
		if err != nil {
			return mcp.NewToolResultError("trading_symbol is required"), nil
		}

		quote, err := kiteClient.GetQuote(&ctx, exchange, tradingSymbol)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get quote: %v", err)), nil
		}

		resultBytes, _ := json.Marshal(quote)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Last Price tool
	lastPriceTool := mcp.NewTool(fmt.Sprintf("kite_get_last_price_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get last price for a trading symbol for user %s", userID)),
		mcp.WithString("exchange", mcp.Description("Exchange (e.g., NSE, BSE, NFO, BFO)"), mcp.Required()),
		mcp.WithString("trading_symbol", mcp.Description("Trading symbol (e.g., RELIANCE, NIFTY24DEC24000CE)"), mcp.Required()),
	)
	srv.AddTool(lastPriceTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		exchange, err := request.RequireString("exchange")
		if err != nil {
			return mcp.NewToolResultError("exchange is required"), nil
		}

		tradingSymbol, err := request.RequireString("trading_symbol")
		if err != nil {
			return mcp.NewToolResultError("trading_symbol is required"), nil
		}

		price, err := kiteClient.GetLastPrice(&ctx, exchange, tradingSymbol)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get last price: %v", err)), nil
		}

		result := map[string]interface{}{"last_price": price}
		resultBytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Place Order tool
	placeOrderTool := mcp.NewTool(fmt.Sprintf("kite_place_order_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Place a new order for user %s", userID)),
		mcp.WithString("exchange", mcp.Description("Exchange (e.g., NSE, BSE, NFO, BFO)"), mcp.Required()),
		mcp.WithString("trading_symbol", mcp.Description("Trading symbol"), mcp.Required()),
		mcp.WithNumber("quantity", mcp.Description("Order quantity"), mcp.Required()),
		mcp.WithNumber("price", mcp.Description("Order price (0 for market orders)"), mcp.Required()),
		mcp.WithString("transaction_type", mcp.Description("BUY or SELL"), mcp.Enum("BUY", "SELL"), mcp.Required()),
		mcp.WithString("product", mcp.Description("Product type (MIS, CNC, NRML)"), mcp.Enum("MIS", "CNC", "NRML"), mcp.Required()),
		mcp.WithString("order_type", mcp.Description("Order type (MARKET, LIMIT, SL, SL-M)"), mcp.Enum("MARKET", "LIMIT", "SL", "SL-M"), mcp.Required()),
		mcp.WithNumber("market_protection_percentage", mcp.Description("Market protection percentage (optional)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("tick_size", mcp.Description("Tick size (optional)"), mcp.DefaultNumber(0.05)),
	)
	srv.AddTool(placeOrderTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		order := &kite.Order{}

		exchange, err := request.RequireString("exchange")
		if err != nil {
			return mcp.NewToolResultError("exchange is required"), nil
		}
		order.Exchange = exchange

		tradingSymbol, err := request.RequireString("trading_symbol")
		if err != nil {
			return mcp.NewToolResultError("trading_symbol is required"), nil
		}
		order.TradingSymbol = tradingSymbol

		quantity, err := request.RequireFloat("quantity")
		if err != nil {
			return mcp.NewToolResultError("quantity is required"), nil
		}
		order.Quantity = quantity

		price, err := request.RequireFloat("price")
		if err != nil {
			return mcp.NewToolResultError("price is required"), nil
		}
		order.Price = price

		transactionType, err := request.RequireString("transaction_type")
		if err != nil {
			return mcp.NewToolResultError("transaction_type is required"), nil
		}
		order.TransactionType = transactionType

		product, err := request.RequireString("product")
		if err != nil {
			return mcp.NewToolResultError("product is required"), nil
		}
		order.Product = product

		orderType, err := request.RequireString("order_type")
		if err != nil {
			return mcp.NewToolResultError("order_type is required"), nil
		}
		order.OrderType = orderType

		order.MarketProtectionPercentage = request.GetFloat("market_protection_percentage", 0)
		order.TickSize = request.GetFloat("tick_size", 0.05)

		orderID, err := kiteClient.PlaceOrder(&ctx, order)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to place order: %v", err)), nil
		}

		result := map[string]interface{}{"order_id": orderID, "status": "success"}
		resultBytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Cancel Order tool
	cancelOrderTool := mcp.NewTool(fmt.Sprintf("kite_cancel_order_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Cancel an existing order for user %s", userID)),
		mcp.WithString("order_id", mcp.Description("Order ID to cancel"), mcp.Required()),
	)
	srv.AddTool(cancelOrderTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orderID, err := request.RequireString("order_id")
		if err != nil {
			return mcp.NewToolResultError("order_id is required"), nil
		}

		err = kiteClient.CancelOrder(&ctx, orderID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to cancel order: %v", err)), nil
		}

		result := map[string]interface{}{"status": "success", "message": "Order cancelled successfully"}
		resultBytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Charges tool
	chargesTool := mcp.NewTool(fmt.Sprintf("kite_get_charges_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get brokerage charges for user %s", userID)),
	)
	srv.AddTool(chargesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		charges, err := kiteClient.GetCharges(&ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get charges: %v", err)), nil
		}

		result := map[string]interface{}{"charges": charges}
		resultBytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Search Instruments tool
	searchInstrumentsTool := mcp.NewTool(fmt.Sprintf("kite_search_instruments_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Search instruments by name, symbol, or exchange with pagination for user %s", userID)),
		mcp.WithString("query", mcp.Description("Search query (name, symbol, or partial match)"), mcp.Required()),
		mcp.WithString("exchange", mcp.Description("Filter by exchange (NSE, BSE, NFO, BFO, MCX) - optional")),
		mcp.WithString("instrument_type", mcp.Description("Filter by type (EQ, FUT, CE, PE) - optional")),
		mcp.WithNumber("limit", mcp.Description("Maximum results to return (default: 50, max: 500)"), mcp.DefaultNumber(50)),
		mcp.WithNumber("offset", mcp.Description("Number of results to skip (default: 0)"), mcp.DefaultNumber(0)),
	)
	srv.AddTool(searchInstrumentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, _ := request.RequireString("query")
		exchange := request.GetString("exchange", "")
		instrumentType := request.GetString("instrument_type", "")
		limit := int(request.GetFloat("limit", 50))
		offset := int(request.GetFloat("offset", 0))

		if limit > 500 {
			limit = 500
		}

		results := searchInstruments(query, exchange, instrumentType, limit, offset)
		resultBytes, _ := json.Marshal(map[string]interface{}{
			"query":   query,
			"results": results,
			"count":   len(results),
			"limit":   limit,
			"offset":  offset,
		})
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Instrument Details tool
	instrumentDetailsTool := mcp.NewTool(fmt.Sprintf("kite_get_instrument_details_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get detailed information about a specific instrument for user %s", userID)),
		mcp.WithString("exchange", mcp.Description("Exchange (e.g., NSE, BSE, NFO, BFO)"), mcp.Required()),
		mcp.WithString("trading_symbol", mcp.Description("Trading symbol"), mcp.Required()),
	)
	srv.AddTool(instrumentDetailsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		exchange, _ := request.RequireString("exchange")
		tradingSymbol, _ := request.RequireString("trading_symbol")

		symbolKey := exchange + ":" + tradingSymbol
		if kite.BrokerInstrumentTokens != nil {
			if instrument, exists := (*kite.BrokerInstrumentTokens)[symbolKey]; exists {
				resultBytes, _ := json.Marshal(instrument)
				return mcp.NewToolResultText(string(resultBytes)), nil
			}
		}
		return mcp.NewToolResultError(fmt.Sprintf("instrument %s not found", symbolKey)), nil
	})

	// Get Historical Data tool
	historicalDataTool := mcp.NewTool(fmt.Sprintf("kite_get_historical_data_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get historical candle data for an instrument for user %s", userID)),
		mcp.WithString("exchange", mcp.Description("Exchange (e.g., NSE, BSE, NFO, BFO)"), mcp.Required()),
		mcp.WithString("trading_symbol", mcp.Description("Trading symbol"), mcp.Required()),
		mcp.WithString("interval", mcp.Description("Candle interval (minute, 3minute, 5minute, 15minute, 30minute, 60minute, day)"), mcp.Required()),
		mcp.WithString("from_date", mcp.Description("Start date (YYYY-MM-DD)"), mcp.Required()),
		mcp.WithString("to_date", mcp.Description("End date (YYYY-MM-DD)"), mcp.Required()),
	)
	srv.AddTool(historicalDataTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		exchange, _ := request.RequireString("exchange")
		tradingSymbol, _ := request.RequireString("trading_symbol")
		interval, _ := request.RequireString("interval")
		fromDate, _ := request.RequireString("from_date")
		toDate, _ := request.RequireString("to_date")

		candles, err := kiteClient.GetHistoricalData(&ctx, exchange, tradingSymbol, interval, fromDate, toDate)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get historical data: %v", err)), nil
		}

		resultBytes, _ := json.Marshal(candles)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Modify Order tool
	modifyOrderTool := mcp.NewTool(fmt.Sprintf("kite_modify_order_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Modify an existing order for user %s", userID)),
		mcp.WithString("order_id", mcp.Description("Order ID to modify"), mcp.Required()),
		mcp.WithString("exchange", mcp.Description("Exchange"), mcp.Required()),
		mcp.WithString("trading_symbol", mcp.Description("Trading symbol"), mcp.Required()),
		mcp.WithNumber("quantity", mcp.Description("New quantity"), mcp.Required()),
		mcp.WithNumber("price", mcp.Description("New price"), mcp.Required()),
		mcp.WithString("transaction_type", mcp.Description("BUY or SELL"), mcp.Enum("BUY", "SELL"), mcp.Required()),
		mcp.WithString("product", mcp.Description("Product type (MIS, CNC, NRML)"), mcp.Enum("MIS", "CNC", "NRML"), mcp.Required()),
		mcp.WithString("order_type", mcp.Description("Order type (MARKET, LIMIT, SL, SL-M)"), mcp.Enum("MARKET", "LIMIT", "SL", "SL-M"), mcp.Required()),
		mcp.WithNumber("market_protection_percentage", mcp.Description("Market protection percentage (optional)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("tick_size", mcp.Description("Tick size (optional)"), mcp.DefaultNumber(0.05)),
	)
	srv.AddTool(modifyOrderTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orderID, _ := request.RequireString("order_id")

		// Create order object with new values
		order := &kite.Order{}
		order.Exchange, _ = request.RequireString("exchange")
		order.TradingSymbol, _ = request.RequireString("trading_symbol")
		order.Quantity, _ = request.RequireFloat("quantity")
		order.Price, _ = request.RequireFloat("price")
		order.TransactionType, _ = request.RequireString("transaction_type")
		order.Product, _ = request.RequireString("product")
		order.OrderType, _ = request.RequireString("order_type")
		order.MarketProtectionPercentage = request.GetFloat("market_protection_percentage", 0)
		order.TickSize = request.GetFloat("tick_size", 0.05)

		err := kiteClient.ModifyOrder(&ctx, orderID, order)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to modify order: %v", err)), nil
		}

		result := map[string]interface{}{"status": "success", "message": "Order modified successfully"}
		resultBytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Option Chain tool
	optionChainTool := mcp.NewTool(fmt.Sprintf("kite_get_option_chain_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get option chain for an underlying instrument for user %s", userID)),
		mcp.WithString("underlying", mcp.Description("Underlying symbol (e.g., NIFTY, BANKNIFTY, RELIANCE)"), mcp.Required()),
		mcp.WithString("expiry", mcp.Description("Expiry date (YYYY-MM-DD) - optional, gets nearest expiry if not provided")),
		mcp.WithNumber("strike_range", mcp.Description("Number of strikes around ATM (default: 10)"), mcp.DefaultNumber(10)),
	)
	srv.AddTool(optionChainTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		underlying, _ := request.RequireString("underlying")
		expiry := request.GetString("expiry", "")
		strikeRange := int(request.GetFloat("strike_range", 10))

		optionChain := getOptionChain(underlying, expiry, strikeRange)
		resultBytes, _ := json.Marshal(optionChain)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Portfolio Holdings tool
	holdingsTool := mcp.NewTool(fmt.Sprintf("kite_get_holdings_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get portfolio holdings for user %s", userID)),
	)
	srv.AddTool(holdingsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		holdings, err := kiteClient.GetHoldings(&ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get holdings: %v", err)), nil
		}

		resultBytes, _ := json.Marshal(holdings)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Get Account Profile tool
	profileTool := mcp.NewTool(fmt.Sprintf("kite_get_profile_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Get account profile information for user %s", userID)),
	)
	srv.AddTool(profileTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		profile, err := kiteClient.GetProfile(&ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get profile: %v", err)), nil
		}

		resultBytes, _ := json.Marshal(profile)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Fetch Instruments tool (kept for backward compatibility)
	instrumentsTool := mcp.NewTool(fmt.Sprintf("kite_fetch_instruments_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Fetch all available instruments for user %s (WARNING: Large dataset, use search instead)", userID)),
	)
	srv.AddTool(instrumentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		instruments, err := kiteClient.FetchInstruments()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch instruments: %v", err)), nil
		}

		// Limit to first 1000 instruments to prevent overwhelming response
		if len(instruments) > 1000 {
			instruments = instruments[:1000]
		}

		result := map[string]interface{}{
			"instruments": instruments,
			"total_count": len(instruments),
			"note":        "Limited to first 1000 instruments. Use kite_search_instruments for specific searches.",
		}
		resultBytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	// Start Data Engine tool (equivalent to Write function)
	dataEngineTool := mcp.NewTool(fmt.Sprintf("kite_start_data_engine_%s", userID),
		mcp.WithDescription(fmt.Sprintf("Start the Kite data collection engine for user %s", userID)),
	)
	srv.AddTool(dataEngineTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		result := map[string]interface{}{
			"status":  "success",
			"message": "Kite data engine started successfully",
		}
		resultBytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultBytes)), nil
	})

	log.Printf("Registered all Kite MCP tools successfully for user: %s", userID)
}

// Helper function to search instruments
func searchInstruments(query, exchange, instrumentType string, limit, offset int) []*kite.Instrument {
	var results []*kite.Instrument

	if kite.BrokerInstrumentTokens == nil {
		return results
	}

	count := 0
	skipped := 0

	for _, instrument := range *kite.BrokerInstrumentTokens {
		// Apply filters
		if exchange != "" && instrument.Exchange != exchange {
			continue
		}
		if instrumentType != "" && instrument.InstrumentType != instrumentType {
			continue
		}

		// Search in name, trading symbol, or token
		queryLower := strings.ToLower(query)
		if strings.Contains(strings.ToLower(instrument.Name), queryLower) ||
			strings.Contains(strings.ToLower(instrument.TradingSymbol), queryLower) ||
			strings.Contains(fmt.Sprintf("%d", instrument.Token), query) {

			// Skip for pagination
			if skipped < offset {
				skipped++
				continue
			}

			// Add to results
			results = append(results, instrument)
			count++

			// Check limit
			if count >= limit {
				break
			}
		}
	}

	return results
}

// Helper function to get option chain
func getOptionChain(underlying, expiry string, strikeRange int) map[string]interface{} {
	if kite.BrokerInstrumentTokens == nil {
		return map[string]interface{}{"error": "instruments not loaded"}
	}

	var underlyingPrice float64 = 0
	var targetExpiry string = expiry

	// Get underlying price from WebSocket if available
	if kiteClient.TickSymbolMap != nil {
		kiteClient.TickSymbolMapMutex.RLock()
		if ticker, exists := kiteClient.TickSymbolMap["NSE:"+underlying]; exists {
			underlyingPrice = ticker.LastPrice
		}
		kiteClient.TickSymbolMapMutex.RUnlock()
	}

	// Find nearest expiry if not provided
	if targetExpiry == "" {
		targetExpiry = findNearestExpiry(underlying)
	}

	// Collect options
	var callOptions []*kite.Instrument
	var putOptions []*kite.Instrument

	for _, instrument := range *kite.BrokerInstrumentTokens {
		if instrument.Name == underlying &&
			(instrument.InstrumentType == "CE" || instrument.InstrumentType == "PE") &&
			instrument.Expiry == targetExpiry {

			if instrument.InstrumentType == "CE" {
				callOptions = append(callOptions, instrument)
			} else {
				putOptions = append(putOptions, instrument)
			}
		}
	}

	// Sort by strike price
	sort.Slice(callOptions, func(i, j int) bool {
		return callOptions[i].Strike < callOptions[j].Strike
	})
	sort.Slice(putOptions, func(i, j int) bool {
		return putOptions[i].Strike < putOptions[j].Strike
	})

	// Filter around ATM if underlying price is available
	if underlyingPrice > 0 && strikeRange > 0 {
		callOptions = filterAroundATM(callOptions, underlyingPrice, strikeRange)
		putOptions = filterAroundATM(putOptions, underlyingPrice, strikeRange)
	}

	return map[string]interface{}{
		"underlying":       underlying,
		"underlying_price": underlyingPrice,
		"expiry":           targetExpiry,
		"call_options":     callOptions,
		"put_options":      putOptions,
		"strike_range":     strikeRange,
	}
}

// Helper function to find nearest expiry
func findNearestExpiry(underlying string) string {
	if kite.BrokerInstrumentTokens == nil {
		return ""
	}

	expiries := make(map[string]bool)
	for _, instrument := range *kite.BrokerInstrumentTokens {
		if instrument.Name == underlying &&
			(instrument.InstrumentType == "CE" || instrument.InstrumentType == "PE") &&
			instrument.Expiry != "" {
			expiries[instrument.Expiry] = true
		}
	}

	// Convert to slice and sort
	var expiryList []string
	for expiry := range expiries {
		expiryList = append(expiryList, expiry)
	}
	sort.Strings(
expiryList)

	// Return nearest future expiry
	now := time.Now().Format("2006-01-02")
	for _, expiry := range expiryList {
		if expiry >= now {
			return expiry
		}
	}

	// If no future expiry found, return the last one
	if len(expiryList) > 0 {
		return expiryList[len(expiryList)-1]
	}

	return ""
}

// Helper function to filter options around ATM
func filterAroundATM(options []*kite.Instrument, atmPrice float64, strikeRange int) []*kite.Instrument {
	if len(options) == 0 {
		return options
	}

	// Find ATM strike index
	atmIndex := 0
	minDiff := math.Abs(options[0].Strike - atmPrice)

	for i, option := range options {
		diff := math.Abs(option.Strike - atmPrice)
		if diff < minDiff {
			minDiff = diff
			atmIndex = i
		}
	}

	// Calculate range
	start := atmIndex - strikeRange
	end := atmIndex + strikeRange + 1

	if start < 0 {
		start = 0
	}
	if end > len(options) {
		end = len(options)
	}

	return options[start:end]
}
