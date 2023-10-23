package orderbook

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

type Match struct {
	Ask        *Order // The asking price from a *seller*
	Bid        *Order // The bidding price from a *buyer*
	SizeFilled float64
	Price      float64
}

// Individual order placed by a trader
type Order struct {
	ID        int64
	Size      float64
	Bid       bool // Bid is a buy order, ask is a sell order
	Limit     *Limit
	Timestamp int64
}

type Orders []*Order

func (o Orders) Len() int           { return len(o) }
func (o Orders) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o Orders) Less(i, j int) bool { return o[i].Timestamp < o[j].Timestamp }

// Creates a new Order
func NewOrder(bid bool, size float64) *Order {
	return &Order{
		ID:        int64(rand.Intn(1000000000000)), // TODO: Implement better ID system then random numbers
		Size:      size,
		Bid:       bid,
		Timestamp: time.Now().UnixNano(),
	}
}

func (o *Order) String() string {
	return fmt.Sprintf("[size: %.2f]", o.Size)
}

func (o *Order) IsFilled() bool {
	return o.Size == 0.0
}

/*
	 A specific price level in the order book. Tracks
		the orders that are at the same price and keeps
		their total volume.
*/
type Limit struct {
	Price       float64
	Orders      Orders
	TotalVolume float64
}

type Limits []*Limit

type ByBestAsk struct{ Limits }

func (a ByBestAsk) Len() int           { return len(a.Limits) }
func (a ByBestAsk) Swap(i, j int)      { a.Limits[i], a.Limits[j] = a.Limits[j], a.Limits[i] }
func (a ByBestAsk) Less(i, j int) bool { return a.Limits[i].Price < a.Limits[j].Price }

type ByBestBid struct{ Limits }

func (b ByBestBid) Len() int           { return len(b.Limits) }
func (b ByBestBid) Swap(i, j int)      { b.Limits[i], b.Limits[j] = b.Limits[j], b.Limits[i] }
func (b ByBestBid) Less(i, j int) bool { return b.Limits[i].Price > b.Limits[j].Price }

// Creates a new Limit with empty list of orders
func NewLimit(price float64) *Limit {
	return &Limit{
		Price:  price,
		Orders: []*Order{},
	}
}

// Adds an order to a specific price level
func (l *Limit) AddOrder(o *Order) {
	o.Limit = l
	l.Orders = append(l.Orders, o)
	l.TotalVolume += o.Size
}

// Removes an order from a specific price level i.e. you want to cancel an order
func (l *Limit) DeleteOrder(o *Order) {
	for i := 0; i < len(l.Orders); i++ {
		if l.Orders[i] == o {
			l.Orders[i] = l.Orders[len(l.Orders)-1]
			l.Orders = l.Orders[:len(l.Orders)-1]
		}
	}

	o.Limit = nil
	l.TotalVolume -= o.Size

	sort.Sort(l.Orders)
}

func (l *Limit) Fill(o *Order) []Match {
	var (
		matches        []Match
		ordersToDelete []*Order
	)

	for _, order := range l.Orders {
		match := l.fillOrder(order, o)
		matches = append(matches, match)

		l.TotalVolume -= match.SizeFilled

		if order.IsFilled() {
			ordersToDelete = append(ordersToDelete, order)
		}

		if o.IsFilled() {
			break
		}
	}

	for _, order := range ordersToDelete {
		l.DeleteOrder(order)
	}

	return matches
}

func (l *Limit) fillOrder(a, b *Order) Match {
	var (
		bid        *Order
		ask        *Order
		sizeFilled float64
	)

	if a.Bid {
		bid = a
		ask = b
	} else {
		bid = b
		ask = a
	}

	if a.Size >= b.Size {
		a.Size -= b.Size
		sizeFilled = b.Size
		b.Size = 0.0
	} else {
		b.Size -= a.Size
		sizeFilled = a.Size
		a.Size = 0.0
	}

	return Match{
		Bid:        bid,
		Ask:        ask,
		SizeFilled: sizeFilled,
		Price:      l.Price,
	}
}

// The entire order book
type Orderbook struct {
	asks []*Limit
	bids []*Limit

	AskLimits map[float64]*Limit
	BidLimits map[float64]*Limit
	Orders    map[int64]*Order //used for api id accessing
}

func NewOrderBook() *Orderbook {
	return &Orderbook{
		asks:      []*Limit{},
		bids:      []*Limit{},
		AskLimits: make(map[float64]*Limit),
		BidLimits: make(map[float64]*Limit),
		Orders:    make(map[int64]*Order),
	}
}

// Always fills the best price. Starts at a certain Limit level until it is completely gone, then it will go ti the next level
func (ob *Orderbook) PlaceMarketOrder(o *Order) []Match {
	// Unless the exchange has no volume,
	matches := []Match{}

	if o.Bid {
		if o.Size > ob.AskTotalVolume() {
			panic(fmt.Errorf("not enough volume [size: %.2f] for market order [size: %.2f]", ob.AskTotalVolume(), o.Size))
		}

		// we use the Asks() func (not the private var) so we get the sorted lists of asks
		for _, limit := range ob.Asks() {
			limitMatches := limit.Fill(o)
			matches = append(matches, limitMatches...)

			if len(limit.Orders) == 0 {
				ob.clearLimit(true, limit)
			}
		}
	} else {
		if o.Size > ob.BidTotalVolume() {
			panic(fmt.Errorf("not enough volume [size: %.2f] for market order [size: %.2f]", ob.BidTotalVolume(), o.Size))
		}

		// we use the Asks() func (not the private var) so we get the sorted lists of asks
		for _, limit := range ob.Bids() {
			limitMatches := limit.Fill(o)
			matches = append(matches, limitMatches...)
			if len(limit.Orders) == 0 {
				ob.clearLimit(true, limit)
			}
		}
	}

	return matches
}

// An order for a specific price point.
// PlaceLimitOrder places a limit order and returns any matches.
func (ob *Orderbook) PlaceLimitOrder(price float64, o *Order) []Match {
	var limit *Limit
	matches := []Match{}

	// If it's a buy order, look for matching sell orders (asks)
	if o.Bid {
		for _, askLimit := range ob.Asks() {
			// Check if the buy order price is greater than or equal to the ask limit price
			if price >= askLimit.Price {
				limitMatches := askLimit.Fill(o)
				matches = append(matches, limitMatches...)

				if len(askLimit.Orders) == 0 {
					ob.clearLimit(true, askLimit) // Clearing ask limit
				}

				if o.IsFilled() {
					break
				}
			}
		}

		limit = ob.BidLimits[price]
	} else { // If it's a sell order, look for matching buy orders (bids)
		for _, bidLimit := range ob.Bids() {
			// Check if the sell order price is less than or equal to the bid limit price
			if price <= bidLimit.Price {
				limitMatches := bidLimit.Fill(o)
				matches = append(matches, limitMatches...)

				if len(bidLimit.Orders) == 0 {
					ob.clearLimit(false, bidLimit) // Clearing bid limit
					fmt.Println("Cleared bid limit")
				}

				if o.IsFilled() {
					break
				}
			}
		}

		limit = ob.AskLimits[price]
	}

	// If the limit wasn't filled and doesn't exist, create it
	if !o.IsFilled() {
		if limit == nil {
			limit = NewLimit(price)

			if o.Bid {
				ob.bids = append(ob.bids, limit)
				ob.BidLimits[price] = limit
			} else {
				ob.asks = append(ob.asks, limit)
				ob.AskLimits[price] = limit
			}
		}
		ob.Orders[o.ID] = o
		limit.AddOrder(o)
	}
	return matches // Return the matches, will be empty if no matches occurred
}

func (ob *Orderbook) clearLimit(bid bool, l *Limit) {
	if bid {
		delete(ob.BidLimits, l.Price)
		for i := 0; i < len(ob.bids); i++ {
			if ob.bids[i] == l {
				ob.bids[i] = ob.bids[len(ob.bids)-1]
				ob.bids = ob.bids[:len(ob.bids)-1]
				break
			}
		}
	} else {
		delete(ob.AskLimits, l.Price)
		for i := 0; i < len(ob.asks); i++ {
			if ob.asks[i] == l {
				ob.asks[i] = ob.asks[len(ob.asks)-1]
				ob.asks = ob.asks[:len(ob.asks)-1]
				break
			}
		}
	}
}

func (ob *Orderbook) CancelOrder(o *Order) {
	limit := o.Limit
	limit.DeleteOrder(o)
	delete(ob.Orders, o.ID)
}

func (ob *Orderbook) BidTotalVolume() float64 {
	totalVolume := 0.0

	for i := 0; i < len(ob.bids); i++ {
		totalVolume += ob.bids[i].TotalVolume
	}

	return totalVolume
}

func (ob *Orderbook) AskTotalVolume() float64 {
	totalVolume := 0.0

	for i := 0; i < len(ob.asks); i++ {
		totalVolume += ob.asks[i].TotalVolume
	}

	return totalVolume
}

func (ob *Orderbook) Asks() []*Limit {
	sort.Sort(ByBestAsk{ob.asks}) // Doesn't return anything, just swaps in memory
	return ob.asks
}

func (ob *Orderbook) Bids() []*Limit {
	sort.Sort(ByBestBid{ob.bids}) // Doesn't return anything, just swaps in memory
	return ob.bids
}
