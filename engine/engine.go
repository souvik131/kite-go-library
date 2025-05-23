package engine

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/souvik131/kite-go-library/kite"
	"github.com/souvik131/kite-go-library/storage"
	"google.golang.org/protobuf/proto"
)

var dateFormatConcise = "20060102"

func Write(ctx *context.Context, k *kite.Kite) {

	Serve(ctx, k)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func readMap(dateStr string) (map[uint32]*storage.TickerMap, error) {

	tokenNameMap := map[uint32]*storage.TickerMap{}
	b, err := os.ReadFile("./binary/map_" + dateStr + ".proto.zstd")
	if err != nil {
		return nil, err
	}
	for len(b) > 8 {
		sizeOfPacket := binary.BigEndian.Uint64(b[0:8])
		packet, err := decompress(b[8 : sizeOfPacket+8])
		if err != nil {
			return nil, err
		}
		b = b[sizeOfPacket+8:]

		data := &storage.Map{}
		err = proto.Unmarshal(packet, data)
		if err != nil {
			return nil, err
		}
		for _, ts := range data.TickerMap {
			tokenNameMap[ts.Token] = ts
		}
	}
	return tokenNameMap, nil
}

func Read(dateStr string) {
	tokenMap, err := readMap(dateStr)
	if err != nil {
		log.Printf("%s", err)
	}

	b, err := os.ReadFile("./binary/market_data_" + dateStr + ".bin.zstd")
	if err != nil {
		log.Printf("%s", err)
	}

	t := &kite.TickerClient{
		TickerChan: make(chan kite.KiteTicker),
	}

	go func() {
		for len(b) > 8 {
			sizeOfPacket := binary.BigEndian.Uint64(b[0:8])
			packet, err := decompress(b[8 : sizeOfPacket+8])
			if err != nil {
				log.Printf("%s", err)
			}
			t.ParseBinary(packet)
			b = b[sizeOfPacket+8:]
		}
	}()
	if err != nil {
		log.Printf("%s", err)
	}
	counter := 0
	start := time.Now()
	timeElapsed := time.Microsecond
	indices := map[string]bool{}
	for {
		select {
		case ticker := <-t.TickerChan:
			counter++
			if t, ok := tokenMap[ticker.Token]; ok {
				ticker.TradingSymbol = t.TradingSymbol
				// if counter%1000000 == 0 {
				fmt.Printf("%v: %+v\n", counter, ticker)
				// }
				indices[t.Name] = true
			}
			timeElapsed = time.Since(start)
		case <-time.After(time.Second):

			keys := make([]string, 0, len(indices))

			for key := range indices {
				keys = append(keys, key)
			}
			fmt.Printf("Read", counter, "F&O records of ("+strings.Join(keys, ", ")+")", "in", timeElapsed)
			log.Panic("exiting")
		}
	}

}

func compress(input []byte) ([]byte, error) {
	var b bytes.Buffer
	bestLevel := zstd.WithEncoderLevel(zstd.SpeedBestCompression)
	encoder, err := zstd.NewWriter(&b, bestLevel)
	if err != nil {
		return nil, err
	}

	_, err = encoder.Write(input)
	if err != nil {
		encoder.Close()
		return nil, err
	}

	err = encoder.Close()
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func decompress(input []byte) ([]byte, error) {
	b := bytes.NewReader(input)
	decoder, err := zstd.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()

	var out bytes.Buffer
	_, err = out.ReadFrom(decoder)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func appendToFile(filename string, data []byte) error {

	compressedData, err := compress(data)
	if err != nil {
		log.Printf("%s", err)
	}

	bytesToSave := make([]byte, 8)
	binary.BigEndian.PutUint64(bytesToSave, uint64(len(compressedData)))
	bytesToSave = append(bytesToSave, compressedData...)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(bytesToSave)
	if err != nil {
		return err
	}
	return nil
}

func saveFile(filePath string, data []byte) error {
	compressedData, err := compress(data)
	if err != nil {
		log.Printf("%s", err)
	}

	bytesToSave := make([]byte, 8)
	binary.BigEndian.PutUint64(bytesToSave, uint64(len(compressedData)))
	log.Println("Saving File : ", binary.BigEndian.Uint16(bytesToSave), uint64(len(compressedData)))
	bytesToSave = append(bytesToSave, compressedData...)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(bytesToSave)
	return err
}

func Serve(ctx *context.Context, k *kite.Kite) {

	tokenTradingsymbolMap := map[uint32]*storage.TickerMap{}

	iMap := &storage.Map{
		TickerMap: []*storage.TickerMap{},
	}

	for name, data := range *kite.BrokerInstrumentTokens {
		tokenTradingsymbolMap[data.Token] = &storage.TickerMap{
			Token:          data.Token,
			TradingSymbol:  strings.Split(name, ":")[1],
			Exchange:       data.Exchange,
			Name:           data.Name,
			Expiry:         data.Expiry,
			Strike:         float32(data.Strike),
			TickSize:       float32(data.TickSize),
			LotSize:        uint32(data.LotSize),
			InstrumentType: data.InstrumentType,
			Segment:        data.Segment,
		}
		iMap.TickerMap = append(iMap.TickerMap, tokenTradingsymbolMap[data.Token])
	}

	bytes, err := proto.Marshal(iMap)
	if err != nil {
		log.Printf("%s", err)
	}

	saveFile("./binary/map_"+time.Now().Format(dateFormatConcise)+".proto.zstd", bytes)

	log.Printf("Instrument Map successfully written to file")

	k.TickerClients = []*kite.TickerClient{}
	// Initialize token tracking
	var processedTokens int64 = 0
	processedSymbols := make(map[string]bool)

	allTokens := []uint32{}
	// Track tokens by segment for different trading hours
	equityTokens := make(map[uint32]bool)
	mcxTokens := make(map[uint32]bool)

	for _, data := range *kite.BrokerInstrumentTokens {
		if data.Exchange == "NSE" || data.Exchange == "NFO" || data.Exchange == "NFO-OPT" || data.Name == "SENSEX" || data.Name == "BANKEX" || data.Segment == "MCX-FUT" {
			allTokens = append(allTokens, data.Token)

			// Track token by segment
			if data.Segment == "MCX-FUT" {
				mcxTokens[data.Token] = true
			} else {
				equityTokens[data.Token] = true
			}
		}
	}
	totalTokens := len(allTokens)
	log.Printf("Total unique tokens to process: %d", totalTokens)

	var symbolsMutex sync.Mutex
	ticker, err := k.GetWebSocketClient(ctx /*, false*/)
	if err != nil {
		log.Printf("%v", err)
	}
	k.TickerClients = append(k.TickerClients, ticker)
	k.TickSymbolMap = map[string]kite.KiteTicker{}

	rotationInterval, err := strconv.ParseFloat(os.Getenv("TA_FEED_TIMEOUT"), 64)
	if err != nil {
		log.Printf("%v", err)
	}
	instrumentsPerRequest, err := strconv.ParseFloat(os.Getenv("TA_FEED_INSTRUMENT_COUNT"), 64)
	if err != nil {
		log.Printf("%v", err)
	}

	// Handle websocket connection
	go func(t *kite.TickerClient) {
		isSubscribed := true
		for range t.ConnectChan {
			log.Printf("Websocket is connected")

			// Start rotation
			go func() {
				for {
					select {
					case <-(*ctx).Done():
						return
					default:

						nowTimeValue := time.Now().Hour()*60 + time.Now().Minute()
						withinEquityTradingTime := 9*60+15 <= nowTimeValue && nowTimeValue <= 15*60+30
						withinMCXTradingTime := 9*60 <= nowTimeValue && nowTimeValue <= 23*60+30
						// Rotate through all tokens in chunks
						for start := 0; start < len(allTokens); start += int(instrumentsPerRequest) {
							select {
							case <-(*ctx).Done():
								return
							default:

								// Function to check if a token is within its trading hours based on segment
								withinTradingTime := func(token uint32) bool {
									if mcxTokens[token] {
										return withinMCXTradingTime
									}
									return withinEquityTradingTime
								}

								// Calculate active tokens count for progress tracking
								activeTokensCount := 0
								if withinEquityTradingTime {
									activeTokensCount += len(equityTokens)
								}
								if withinMCXTradingTime {
									activeTokensCount += len(mcxTokens)
								}
								// Update totalTokens to reflect only active tokens
								totalTokens = activeTokensCount
								end := start + int(instrumentsPerRequest)
								if end > len(allTokens) {
									end = len(allTokens)
								}

								// Unsubscribe from previous chunk
								prevStart := start - int(instrumentsPerRequest)
								if prevStart >= 0 {
									prevEnd := start
									if isSubscribed {
										t.Unsubscribe(ctx, allTokens[prevStart:prevEnd])
									}
									isSubscribed = false
								}

								// Filter current batch for active tokens
								activeBatchTokens := []uint32{}
								for i := start; i < end; i++ {
									if withinTradingTime(allTokens[i]) {
										activeBatchTokens = append(activeBatchTokens, allTokens[i])
									}
								}

								// Subscribe to active tokens in current batch
								if len(activeBatchTokens) > 0 {
									t.SubscribeFull(ctx, activeBatchTokens)
									isSubscribed = true
								} else {
									isSubscribed = false
								}
								// Sleep for rotation interval
								<-time.After(time.Duration(rotationInterval) * time.Second)
							}
						}
					}
				}
			}()
		}
	}(ticker)

	// Handle ticker data
	go func(t *kite.TickerClient) {
		for ticker := range t.TickerChan {
			symbolsMutex.Lock()

			// Populate TickSymbolMap for quote functionality using dedicated mutex
			k.TickSymbolMapMutex.Lock()
			if ticker.TradingSymbol != "" {
				k.TickSymbolMap[ticker.TradingSymbol] = ticker
			}
			// Also store with exchange:symbol format
			if tokenSymbol, exists := kite.TokenSymbolMap[ticker.Token]; exists {
				k.TickSymbolMap[tokenSymbol] = ticker
			}
			k.TickSymbolMapMutex.Unlock()

			if !processedSymbols[ticker.TradingSymbol] {
				processedSymbols[ticker.TradingSymbol] = true
				processed := atomic.AddInt64(&processedTokens, 1)
				if processed%10000 == 0 || float64(processed)/float64(totalTokens) == 1 {
					log.Printf("New token processed: %s (%d/%d - %.2f%%)",
						ticker.TradingSymbol,
						processed,
						totalTokens,
						float64(processed)/float64(totalTokens)*100)
				}
				if float64(processed)/float64(totalTokens) == 1 {
					processedTokens = 0
					processedSymbols = make(map[string]bool)
					// storage.DataMapMutex.Lock()
					// storage.DataMap = map[string]*storage.Ticker{}
					// storage.DataMapMutex.Unlock()
				}
			}
			symbolsMutex.Unlock()
		}
	}(ticker)

	// Handle binary data
	go func(t *kite.TickerClient) {
		for message := range t.BinaryTickerChan {
			nowTimeValue := time.Now().Hour()*60 + time.Now().Minute()
			withinEquityTradingTime := 9*60+15 <= nowTimeValue && nowTimeValue <= 15*60+30
			withinMCXTradingTime := 9*60 <= nowTimeValue && nowTimeValue <= 23*60+30

			// Save data if either equity or MCX is trading
			if withinEquityTradingTime || withinMCXTradingTime {
				appendToFile("./binary/market_data_equity_mcx_"+time.Now().Format(dateFormatConcise)+".bin.zstd", message)
			}

			// data := &storage.Data{
			// 	Tickers: []*storage.Ticker{},
			// }
			numOfPackets := binary.BigEndian.Uint16(message[0:2])
			if numOfPackets > 0 {

				message = message[2:]
				for {
					if numOfPackets == 0 {
						break
					}

					numOfPackets--
					packetSize := binary.BigEndian.Uint16(message[0:2])
					packet := kite.Packet(message[2 : packetSize+2])
					values := packet.ParseBinary(int(math.Min(64, float64(len(packet)))))

					ticker := &storage.Ticker{
						Depth: &storage.Depth{
							Buy:  []*storage.Order{},
							Sell: []*storage.Order{},
						},
					}
					if len(values) >= 2 {
						ticker.Token = values[0]
						ticker.LastPrice = values[1]
					}
					switch len(values) {
					case 2:
					case 7:
						ticker.High = values[2]
						ticker.Low = values[3]
						ticker.Open = values[4]
						ticker.Close = values[5]
						ticker.ExchangeTimestamp = values[6]
					case 8:
						ticker.High = values[2]
						ticker.Low = values[3]
						ticker.Open = values[4]
						ticker.Close = values[5]
						ticker.PriceChange = values[6]
						ticker.ExchangeTimestamp = values[7]
					case 11:
						ticker.LastTradedQuantity = values[2]
						ticker.AverageTradedPrice = values[3]
						ticker.VolumeTraded = values[4]
						ticker.TotalBuy = values[5]
						ticker.TotalSell = values[6]
						ticker.High = values[7]
						ticker.Low = values[8]
						ticker.Open = values[9]
						ticker.Close = values[10]
					case 16:
						ticker.LastTradedQuantity = values[2]
						ticker.AverageTradedPrice = values[3]
						ticker.VolumeTraded = values[4]
						ticker.TotalBuy = values[5]
						ticker.TotalSell = values[6]
						ticker.High = values[7]
						ticker.Low = values[8]
						ticker.Open = values[9]
						ticker.Close = values[10]
						ticker.LastTradedTimestamp = values[11]
						ticker.OI = values[12]
						ticker.OIHigh = values[13]
						ticker.OILow = values[14]
						ticker.ExchangeTimestamp = values[15]
					default:
						log.Printf("unkown length of packet", len(values), values)
					}

					if len(packet) > 64 {

						packet = packet[64:]

						values := packet.ParseMarketDepth()
						lobDepth := len(values) / 6

						for {
							if len(values) == 0 {

								break
							}
							qty := values[0]
							price := values[1]
							orders := values[2]
							if len(ticker.Depth.Buy) < lobDepth {
								ticker.Depth.Buy = append(ticker.Depth.Buy, &storage.Order{Price: price, Quantity: qty, Orders: orders})
							} else {

								ticker.Depth.Sell = append(ticker.Depth.Sell, &storage.Order{Price: price, Quantity: qty, Orders: orders})
							}
							values = values[3:]

						}
					}
					if len(message) > int(packetSize+2) {
						message = message[packetSize+2:]
					}

					tokenData := tokenTradingsymbolMap[ticker.Token]
					ticker.LotSize = tokenData.LotSize
					_, err := json.Marshal(ticker)
					if err != nil {
						log.Printf("%s", err)
					}

				}
			}

		}
	}(ticker)

	// Start serving
	go ticker.Serve(ctx)
	log.Printf("Websocket service started")

	// Wait for context cancellation
	<-(*ctx).Done()
	log.Printf("Shutting down websocket service")

}
