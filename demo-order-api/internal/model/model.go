// Package model defines the domain types for the order processing service.
package model

import "time"

// Order represents a customer order.
type Order struct {
	ID             string    `json:"id" dynamodbav:"id"`
	Items          []Item    `json:"items" dynamodbav:"items"`
	Subtotal       float64   `json:"subtotal" dynamodbav:"subtotal"`
	DiscountCode   string    `json:"discount_code,omitempty" dynamodbav:"discount_code,omitempty"`
	DiscountAmount float64   `json:"discount_amount" dynamodbav:"discount_amount"`
	Total          float64   `json:"total" dynamodbav:"total"`
	Status         string    `json:"status" dynamodbav:"status"`
	CreatedAt      time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

// Item represents a line item within an order.
type Item struct {
	Name      string  `json:"name" dynamodbav:"name"`
	Quantity  int     `json:"quantity" dynamodbav:"quantity"`
	UnitPrice float64 `json:"unit_price" dynamodbav:"unit_price"`
}

// DiscountTier defines a named discount percentage.
type DiscountTier struct {
	Name string  `json:"name"`
	Rate float64 `json:"rate"`
}

// CreateOrderRequest is the payload for POST /orders.
type CreateOrderRequest struct {
	Items        []Item `json:"items"`
	DiscountCode string `json:"discount_code,omitempty"`
}

// Order statuses.
const (
	StatusPending   = "pending"
	StatusConfirmed = "confirmed"
	StatusRefunded  = "refunded"
)
