// Package store provides DynamoDB operations for orders.
package store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/example/demo-incident-response/demo-order-api/internal/model"
)

const (
	tableName   = "demo-orders"
	statusIndex = "status-index"
)

// OrderStore handles DynamoDB operations for orders.
type OrderStore struct {
	client *dynamodb.Client
}

// New creates an OrderStore with the given DynamoDB client.
func New(client *dynamodb.Client) *OrderStore {
	return &OrderStore{client: client}
}

// Create writes a new order to DynamoDB.
func (s *OrderStore) Create(ctx context.Context, order model.Order) error {
	item, err := attributevalue.MarshalMap(order)
	if err != nil {
		return fmt.Errorf("marshalling order: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("putting order: %w", err)
	}
	return nil
}

// Get retrieves an order by ID.
func (s *OrderStore) Get(ctx context.Context, id string) (*model.Order, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting order: %w", err)
	}
	if out.Item == nil {
		return nil, nil
	}

	var order model.Order
	if err := attributevalue.UnmarshalMap(out.Item, &order); err != nil {
		return nil, fmt.Errorf("unmarshalling order: %w", err)
	}
	return &order, nil
}

// List returns all orders, optionally filtered by status.
func (s *OrderStore) List(ctx context.Context, status string) ([]model.Order, error) {
	if status != "" {
		return s.listByStatus(ctx, status)
	}
	return s.scanAll(ctx)
}

func (s *OrderStore) listByStatus(ctx context.Context, status string) ([]model.Order, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String(statusIndex),
		KeyConditionExpression: aws.String("#s = :status"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: status},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("querying orders by status: %w", err)
	}

	var orders []model.Order
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &orders); err != nil {
		return nil, fmt.Errorf("unmarshalling orders: %w", err)
	}
	return orders, nil
}

func (s *OrderStore) scanAll(ctx context.Context) ([]model.Order, error) {
	out, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("scanning orders: %w", err)
	}

	var orders []model.Order
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &orders); err != nil {
		return nil, fmt.Errorf("unmarshalling orders: %w", err)
	}
	return orders, nil
}

// Update writes an updated order back to DynamoDB.
func (s *OrderStore) Update(ctx context.Context, order model.Order) error {
	return s.Create(ctx, order)
}
