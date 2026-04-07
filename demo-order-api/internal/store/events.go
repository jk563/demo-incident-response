package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// AgentEvent represents a single triage agent event from the observer table.
type AgentEvent struct {
	IncidentID string         `dynamodbav:"incident_id" json:"incident_id"`
	Seq        int            `dynamodbav:"seq" json:"seq"`
	EventType  string         `dynamodbav:"event_type" json:"event_type"`
	Timestamp  string         `dynamodbav:"timestamp" json:"timestamp"`
	Detail     map[string]any `dynamodbav:"detail" json:"detail"`
}

// EventStore handles DynamoDB operations for agent observer events.
type EventStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewEventStore creates an EventStore with the given DynamoDB client and table name.
func NewEventStore(client *dynamodb.Client, tableName string) *EventStore {
	return &EventStore{client: client, tableName: tableName}
}

// LatestIncident returns the incident ID of the most recent agent invocation.
func (s *EventStore) LatestIncident(ctx context.Context) (string, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"incident_id": &types.AttributeValueMemberS{Value: "_latest"},
			"seq":         &types.AttributeValueMemberN{Value: "0"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("getting latest incident: %w", err)
	}
	if out.Item == nil {
		return "", nil
	}

	var result struct {
		IncidentRef string `dynamodbav:"incident_ref"`
	}
	if err := attributevalue.UnmarshalMap(out.Item, &result); err != nil {
		return "", fmt.Errorf("unmarshalling latest incident: %w", err)
	}
	return result.IncidentRef, nil
}

// IncidentSummary represents a past incident for the dropdown list.
type IncidentSummary struct {
	IncidentID string `dynamodbav:"incident_ref" json:"incident_id"`
	AlarmName  string `dynamodbav:"alarm_name" json:"alarm_name"`
	StartedAt  string `dynamodbav:"started_at" json:"started_at"`
}

// ListIncidents returns recent incidents, newest first.
func (s *EventStore) ListIncidents(ctx context.Context) ([]IncidentSummary, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("incident_id = :idx"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":idx": &types.AttributeValueMemberS{Value: "_incidents"},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(20),
	})
	if err != nil {
		return nil, fmt.Errorf("querying incidents: %w", err)
	}

	var incidents []IncidentSummary
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &incidents); err != nil {
		return nil, fmt.Errorf("unmarshalling incidents: %w", err)
	}
	return incidents, nil
}

// ListEvents returns events for an incident after the given sequence number.
func (s *EventStore) ListEvents(ctx context.Context, incidentID string, afterSeq int) ([]AgentEvent, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("incident_id = :id AND seq > :after"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id":    &types.AttributeValueMemberS{Value: incidentID},
			":after": &types.AttributeValueMemberN{Value: strconv.Itoa(afterSeq)},
		},
		ScanIndexForward: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}

	var events []AgentEvent
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &events); err != nil {
		return nil, fmt.Errorf("unmarshalling events: %w", err)
	}
	return events, nil
}
