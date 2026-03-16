// Package client provides a shared HTTP client for the order API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

// Order mirrors the API response.
type Order struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	Total          float64 `json:"total"`
	DiscountCode   string  `json:"discount_code,omitempty"`
	DiscountAmount float64 `json:"discount_amount"`
}

// Item is a line item in a create request.
type Item struct {
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// CreateRequest is the POST /orders payload.
type CreateRequest struct {
	Items        []Item `json:"items"`
	DiscountCode string `json:"discount_code,omitempty"`
}

// Client wraps HTTP calls to the order API.
type Client struct {
	base string
	http *http.Client
}

// New creates a client targeting the given base URL.
func New(base string) *Client {
	return &Client{
		base: base,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// CreateOrder sends POST /orders and returns the created order.
func (c *Client) CreateOrder(req CreateRequest) (*Order, int, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal: %w", err)
	}
	resp, err := c.http.Post(c.base+"/orders", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, b)
	}

	var order Order
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode: %w", err)
	}
	return &order, resp.StatusCode, nil
}

// GetOrder sends GET /orders/{id}.
func (c *Client) GetOrder(id string) (*Order, int, error) {
	resp, err := c.http.Get(c.base + "/orders/" + id)
	if err != nil {
		return nil, 0, fmt.Errorf("get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, b)
	}

	var order Order
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode: %w", err)
	}
	return &order, resp.StatusCode, nil
}

// ListOrders sends GET /orders.
func (c *Client) ListOrders() ([]Order, int, error) {
	resp, err := c.http.Get(c.base + "/orders")
	if err != nil {
		return nil, 0, fmt.Errorf("get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, b)
	}

	var orders []Order
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode: %w", err)
	}
	return orders, resp.StatusCode, nil
}

// RefundOrder sends POST /orders/{id}/refund.
func (c *Client) RefundOrder(id string) (*Order, int, error) {
	resp, err := c.http.Post(c.base+"/orders/"+id+"/refund", "application/json", nil)
	if err != nil {
		return nil, 0, fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, b)
	}

	var order Order
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode: %w", err)
	}
	return &order, resp.StatusCode, nil
}

// Sample product catalogue for random order generation.
var products = []Item{
	{Name: "Mechanical Keyboard", Quantity: 1, UnitPrice: 89.99},
	{Name: "USB-C Cable", Quantity: 2, UnitPrice: 9.99},
	{Name: "Monitor Stand", Quantity: 1, UnitPrice: 45.00},
	{Name: "Webcam HD", Quantity: 1, UnitPrice: 59.99},
	{Name: "Mouse Pad XL", Quantity: 1, UnitPrice: 19.99},
	{Name: "Headset", Quantity: 1, UnitPrice: 129.99},
	{Name: "Desk Lamp", Quantity: 1, UnitPrice: 34.99},
	{Name: "Notebook", Quantity: 3, UnitPrice: 4.99},
	{Name: "Pen Set", Quantity: 1, UnitPrice: 12.50},
	{Name: "Cable Tidy", Quantity: 2, UnitPrice: 7.99},
}

// RandomItems returns 1-3 random items from the catalogue.
func RandomItems() []Item {
	n := rand.IntN(3) + 1
	picked := make([]Item, 0, n)
	for _, i := range rand.Perm(len(products))[:n] {
		item := products[i]
		item.Quantity = rand.IntN(3) + 1
		picked = append(picked, item)
	}
	return picked
}
