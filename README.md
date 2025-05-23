# Kite Go Library

A comprehensive Go library for Zerodha Kite trading platform integration with support for API trading, real-time data collection, binary storage, and Model Context Protocol (MCP) integration.

## Features

- **Trading API Integration**: Complete Kite Connect API wrapper for trading operations
- **Real-time Data Collection**: WebSocket-based market data streaming and storage
- **Binary Data Storage**: Compressed storage of NSE/NFO/MCX market data with protobuf serialization
- **MCP Server**: Claude Desktop integration for AI-powered trading assistance
- **Web Interface**: HTTP API endpoints for web-based trading applications
- **Multi-Exchange Support**: NSE, BSE, NFO, BFO, MCX market data and trading

## Quick Start

### 1. Configuration

Copy the example environment file and configure your credentials:

```bash
cp .env_example .env
```

Edit `.env` with your Kite credentials:

```env
TA_ID=your_user_id                    # Kite Username
TA_PASSWORD=your_password             # Kite Password
TA_TOTP=your_totpkey                  # Kite TOTP Secret (not OTP)
TA_APIKEY=your_api_key                # API key from kite.trade
TA_APISECRET=your_api_secret          # API secret from kite.trade
TA_PATH=http://127.0.0.1:80/kite      # API path for web mode
TA_PORT=80                            # Port for web server
TA_LOGINTYPE=WEB                      # Login type: WEB or API
TA_FEED_TIMEOUT=2                     # Data feed rotation interval (seconds)
TA_FEED_INSTRUMENT_COUNT=3000         # Instruments per WebSocket batch
```

### 2. Docker Deployment

```bash
docker-compose up -d
```

### 3. Direct Execution

```bash
# Build the project
go build -o kite-mcp-server

# Run as MCP server
./kite-mcp-server

# Or run with environment variables
TA_LOGINTYPE=API ./kite-mcp-server
```

## Usage Modes

### 1. API Mode (`TA_LOGINTYPE=API`)

Direct API integration for programmatic trading:

```go
import "github.com/souvik131/kite-go-library/kite"

ctx := context.Background()
kiteClient := &kite.Kite{}

// Login
err := kiteClient.Login(&ctx)

// Place order
order := &kite.Order{
    Exchange: "NSE",
    TradingSymbol: "RELIANCE",
    Quantity: 1,
    Price: 2500.0,
    TransactionType: "BUY",
    Product: "CNC",
    OrderType: "LIMIT",
}
orderID, err := kiteClient.PlaceOrder(&ctx, order)

// Get positions
err = kiteClient.GetPositions(&ctx)

// Get quotes
quote, err := kiteClient.GetQuote(&ctx, "NSE", "RELIANCE")
```

### 2. Web Mode (`TA_LOGINTYPE=WEB`)

HTTP server with REST API endpoints for web applications. Access trading functions via HTTP requests to configured path and port.

### 3. MCP Server Mode

Model Context Protocol server for Claude Desktop integration, providing AI-powered trading assistance.

## Binary Data Storage

The library automatically collects and stores market data in compressed binary format:

### Storage Structure

```
binary/
├── map_YYYYMMDD.proto.zstd           # Instrument mapping (token -> symbol)
├── market_data_equity_mcx_YYYYMMDD.bin.zstd  # Market data (equity + MCX)
```

### Data Collection Features

- **Real-time Collection**: Streams live market data via WebSocket
- **Compression**: Zstandard compression for optimal storage
- **Rotation**: Automatic instrument rotation based on `TA_FEED_TIMEOUT`
- **Trading Hours**: Respects market timings (Equity: 9:15-15:30, MCX: 9:00-23:30)
- **Multi-Exchange**: NSE, NFO, MCX data collection

### Reading Stored Data

```go
import "github.com/souvik131/kite-go-library/engine"

// Read historical data for a specific date
engine.Read("20240115") // YYYYMMDD format
```

## MCP Integration with Claude Desktop

### Setup

1. **Build the MCP server**:

```bash
go build -o kite-mcp-server
```

2. **Configure Claude Desktop**:

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "kite-server": {
      "command": "./kite-mcp-server",
      "args": [],
      "env": {
        "TA_ID": "your_user_id",
        "TA_PASSWORD": "your_password",
        "TA_TOTP": "your_totpkey",
        "TA_API_KEY": "your_api_key",
        "TA_API_SECRET": "your_api_secret",
        "TA_LOGINTYPE": "API",
        "TA_PATH": "http://127.0.0.1:80/kite",
        "TA_PORT": "80",
        "TA_FEED_TIMEOUT": "2",
        "TA_FEED_INSTRUMENT_COUNT": "3000"
      }
    }
  }
}
```

3. **Restart Claude Desktop** to load the MCP server.

### Available MCP Tools

The MCP server provides the following tools (replace `{user_id}` with your actual user ID):

#### Account Management

- `kite_get_margin_{user_id}` - Get account margin details
- `kite_get_profile_{user_id}` - Get account profile information
- `kite_get_holdings_{user_id}` - Get portfolio holdings
- `kite_get_charges_{user_id}` - Get brokerage charges

#### Trading Operations

- `kite_place_order_{user_id}` - Place new orders
- `kite_modify_order_{user_id}` - Modify existing orders
- `kite_cancel_order_{user_id}` - Cancel orders
- `kite_get_orders_{user_id}` - Get all orders
- `kite_get_order_history_{user_id}` - Get order history
- `kite_get_positions_{user_id}` - Get current positions

#### Market Data

- `kite_get_quote_{user_id}` - Get real-time quotes
- `kite_get_last_price_{user_id}` - Get last traded price
- `kite_get_historical_data_{user_id}` - Get historical candle data
- `kite_get_option_chain_{user_id}` - Get option chain data

#### Instrument Search

- `kite_search_instruments_{user_id}` - Search instruments with filters
- `kite_get_instrument_details_{user_id}` - Get detailed instrument info
- `kite_fetch_instruments_{user_id}` - Fetch all instruments (limited)

#### Data Engine

- `kite_start_data_engine_{user_id}` - Start real-time data collection

### Example MCP Usage in Claude

```
"Get my current positions and margin"
"Place a buy order for 10 shares of RELIANCE at market price"
"Search for NIFTY options expiring this week"
"Get historical data for BANKNIFTY for the last 5 days"
"Show me the option chain for NIFTY"
```

## API Reference

### Core Trading Functions

#### Authentication

```go
Login(ctx *context.Context) error
```

#### Order Management

```go
PlaceOrder(ctx *context.Context, order *Order) (string, error)
ModifyOrder(ctx *context.Context, orderId string, order *Order) error
CancelOrder(ctx *context.Context, orderId string) error
GetOrders(ctx *context.Context) ([]*OrderStatus, error)
GetOrderHistory(ctx *context.Context, orderId string) ([]*OrderStatus, error)
```

#### Portfolio & Positions

```go
GetPositions(ctx *context.Context) error
GetHoldings(ctx *context.Context) ([]*Holding, error)
GetMargin(ctx *context.Context) (*Margin, error)
```

#### Market Data

```go
GetQuote(ctx *context.Context, exchange string, tradingSymbol string) (*Quote, error)
GetLastPrice(ctx *context.Context, exchange string, tradingSymbol string) (float64, error)
GetHistoricalData(ctx *context.Context, exchange, symbol, interval, from, to string) ([]*Candle, error)
```

#### WebSocket Streaming

```go
SubscribeLTP(ctx *context.Context, tokens []string) error
SubscribeFull(ctx *context.Context, tokens []string) error
SubscribeQuote(ctx *context.Context, tokens []string) error
Unsubscribe(ctx *context.Context, tokens []string) error
```

### Order Structure

```go
type Order struct {
    Exchange                    string  `json:"exchange"`
    TradingSymbol              string  `json:"tradingsymbol"`
    Quantity                   float64 `json:"quantity"`
    Price                      float64 `json:"price"`
    TransactionType            string  `json:"transaction_type"` // BUY, SELL
    Product                    string  `json:"product"`          // MIS, CNC, NRML
    OrderType                  string  `json:"order_type"`       // MARKET, LIMIT, SL, SL-M
    MarketProtectionPercentage float64 `json:"market_protection_percentage"`
    TickSize                   float64 `json:"tick_size"`
}
```

## Configuration Options

| Variable                   | Description                      | Default | Required |
| -------------------------- | -------------------------------- | ------- | -------- |
| `TA_ID`                    | Kite username                    | -       | Yes      |
| `TA_PASSWORD`              | Kite password                    | -       | Yes      |
| `TA_TOTP`                  | TOTP secret key                  | -       | Yes      |
| `TA_APIKEY`                | Kite API key                     | -       | Yes      |
| `TA_APISECRET`             | Kite API secret                  | -       | Yes      |
| `TA_LOGINTYPE`             | Login mode (WEB/API)             | WEB     | No       |
| `TA_PATH`                  | Web server path                  | /kite   | No       |
| `TA_PORT`                  | Web server port                  | 80      | No       |
| `TA_FEED_TIMEOUT`          | Data rotation interval (seconds) | 2       | No       |
| `TA_FEED_INSTRUMENT_COUNT` | Instruments per batch            | 3000    | No       |

## Trading Hours

The system respects market trading hours:

- **Equity Markets (NSE/BSE)**: 9:15 AM - 3:30 PM
- **F&O Markets (NFO)**: 9:15 AM - 3:30 PM
- **Commodity Markets (MCX)**: 9:00 AM - 11:30 PM

Data collection automatically starts/stops based on these timings.

## File Structure

```
├── main.go                 # MCP server entry point
├── engine/                 # Data collection engine
│   └── engine.go          # WebSocket data processing
├── kite/                  # Core Kite API library
│   ├── kite_login.go      # Authentication
│   ├── kite_place_order.go # Order management
│   ├── kite_get_*.go      # Data retrieval functions
│   ├── kite_ws.go         # WebSocket client
│   └── models.go          # Data structures
├── storage/               # Binary storage
│   ├── feed_store.proto   # Protobuf definitions
│   └── feed_store.pb.go   # Generated protobuf code
├── web/                   # Web interface
├── ws/                    # WebSocket utilities
└── binary/                # Stored market data
```

## Building from Source

```bash
# Clone repository
git clone <repository-url>
cd kite-go-library

# Install dependencies
go mod download

# Build MCP server
go build -o kite-mcp-server

# Build with specific tags (if needed)
go build -tags production -o kite-mcp-server
```

## Docker Support

```yaml
# docker-compose.yml
version: "3.8"
services:
  kite-server:
    build: .
    ports:
      - "80:80"
    environment:
      - TA_LOGINTYPE=WEB
    env_file:
      - .env
    volumes:
      - ./binary:/app/binary
```

## License

This project is licensed under the MIT License.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## Support

For issues and questions:

- Create an issue on GitHub
- Check existing documentation
- Review the example configurations

## Disclaimer

This library is for educational and development purposes. Always test thoroughly before using in production trading environments. Trading involves financial risk.
