package orderbook

import (
	"fmt"
	"reflect"
	"testing"
)

func assert(t *testing.T, a, b any) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("%+v != %+v", a, b)
	}
}

func TestLimit(t *testing.T) {
	l := NewLimit(10_000)
	buyOrderA := NewOrder(true, 5)
	buyOrderB := NewOrder(true, 8)
	buyOrderC := NewOrder(true, 10)

	l.AddOrder(buyOrderA)
	l.AddOrder(buyOrderB)
	l.AddOrder(buyOrderC)

	l.DeleteOrder(buyOrderB)

	fmt.Println(l)
}

func TestPlaceLimitOrder(t *testing.T) {
	ob := NewOrderBook()

	sellOrderA := NewOrder(false, 10)
	sellOrderB := NewOrder(false, 5)
	ob.PlaceLimitOrder(10_000, sellOrderA)
	ob.PlaceLimitOrder(9_000, sellOrderB)

	assert(t, len(ob.Orders), 2)
	assert(t, ob.Orders[sellOrderA.ID], sellOrderA)
	assert(t, ob.Orders[sellOrderB.ID], sellOrderB)
	assert(t, len(ob.asks), 2)
}

func TestPlaceMarketOrder(t *testing.T) {
	ob := NewOrderBook()

	sellOrder := NewOrder(false, 20)
	ob.PlaceLimitOrder(10_000, sellOrder)

	buyOrder := NewOrder(true, 10)
	matches := ob.PlaceMarketOrder(buyOrder)

	assert(t, len(matches), 1)
	assert(t, len(ob.asks), 1)
	assert(t, ob.AskTotalVolume(), 10.0)
	assert(t, matches[0].Ask, sellOrder)
	assert(t, matches[0].Bid, buyOrder)
	assert(t, matches[0].SizeFilled, 10.0)
	assert(t, matches[0].Price, 10_000.0)
	assert(t, buyOrder.IsFilled(), true)
}

func TestPlaceMarketOrderMultiFill(t *testing.T) {
	ob := NewOrderBook()

	buyOrderA := NewOrder(true, 5)
	buyOrderB := NewOrder(true, 8)
	buyOrderC := NewOrder(true, 10)
	buyOrderD := NewOrder(true, 1)

	ob.PlaceLimitOrder(5_000, buyOrderC)
	ob.PlaceLimitOrder(5_000, buyOrderD)
	ob.PlaceLimitOrder(9_000, buyOrderB)
	ob.PlaceLimitOrder(10_000, buyOrderA)

	assert(t, ob.BidTotalVolume(), 24.00)

	sellOrder := NewOrder(false, 20)
	matches := ob.PlaceMarketOrder(sellOrder)

	assert(t, ob.BidTotalVolume(), 4.0)
	assert(t, len(matches), 3)
	assert(t, len(ob.bids), 1)
}

func TestPlaceLimitOrderMultiFill(t *testing.T) {
	ob := NewOrderBook()

	// Place three buy limit orders
	buyOrderA := NewOrder(true, 8) // Buy 3 at 10,000
	buyOrderB := NewOrder(true, 6) // Buy 4 at 10,000
	buyOrderC := NewOrder(true, 3) // Buy 3 at 10,000

	ob.PlaceLimitOrder(10_000, buyOrderA)
	ob.PlaceLimitOrder(9_000, buyOrderB)
	ob.PlaceLimitOrder(8_000, buyOrderC)

	assert(t, ob.BidTotalVolume(), 17.0) // Total buy volume should be 10

	// Place a sell limit order
	sellOrder := NewOrder(false, 10) // Sell 10 at 10,000
	matches := ob.PlaceLimitOrder(9_000, sellOrder)

	// Check that the sell order was matched with all the buy orders
	assert(t, len(matches), 2)            // There should be three matches
	assert(t, ob.BidTotalVolume(), 7.0)   // All buy orders should be filled, so volume is 0
	assert(t, sellOrder.IsFilled(), true) // Sell order should be fully filled
	assert(t, matches[0].Price, 10_000.0) // Price should jump to highest of 10,000
	assert(t, matches[0].SizeFilled, 8.0) // 8 of the 10 sell order volume should be filled
	assert(t, matches[1].Price, 9_000.0)  // Price should jump to 9_000 for remaining
}

func TestCancelOrder(t *testing.T) {
	ob := NewOrderBook()
	buyOrder := NewOrder(true, 4)
	ob.PlaceLimitOrder(10_000, buyOrder)

	assert(t, ob.BidTotalVolume(), 4.0)

	ob.CancelOrder(buyOrder)
	assert(t, ob.BidTotalVolume(), 0.0)

	_, ok := ob.Orders[buyOrder.ID]
	assert(t, ok, false)
}
