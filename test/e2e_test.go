package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var targetURL string

func TestMain(m *testing.M) {
	targetURL = os.Getenv("TEST_TARGET")
	if targetURL == "" {
		targetURL = "http://localhost:8080"
	}
	targetURL = strings.TrimRight(targetURL, "/")
	os.Exit(m.Run())
}

// Order mirrors the API response structure.
type Order struct {
	ID             string  `json:"id"`
	Items          []Item  `json:"items"`
	Subtotal       float64 `json:"subtotal"`
	DiscountCode   string  `json:"discount_code,omitempty"`
	DiscountAmount float64 `json:"discount_amount"`
	Total          float64 `json:"total"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// Item mirrors the API item structure.
type Item struct {
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// CreateRequest is the payload for POST /orders.
type CreateRequest struct {
	Items        []Item `json:"items"`
	DiscountCode string `json:"discount_code,omitempty"`
}

var client = &http.Client{Timeout: 10 * time.Second}

func doPost(t *testing.T, path string, body any) (*http.Response, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(data)
	}
	resp, err := client.Post(targetURL+path, "application/json", r)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, b
}

func doGet(t *testing.T, path string) (*http.Response, []byte) {
	t.Helper()
	resp, err := client.Get(targetURL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, b
}

func decodeOrder(t *testing.T, data []byte) Order {
	t.Helper()
	var o Order
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatalf("decode order: %v\nbody: %s", err, data)
	}
	return o
}

func createTestOrder(t *testing.T, items []Item, discountCode string) Order {
	t.Helper()
	req := CreateRequest{Items: items, DiscountCode: discountCode}
	resp, body := doPost(t, "/orders", req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	return decodeOrder(t, body)
}

func TestHealth(t *testing.T) {
	resp, body := doGet(t, "/health")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(strings.ToLower(string(body)), "healthy") {
		t.Fatalf("expected body to contain 'healthy', got: %s", body)
	}
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name         string
		items        []Item
		discountCode string
		wantSubtotal float64
		wantDiscount float64
		wantTotal    float64
	}{
		{
			name:         "no discount",
			items:        []Item{{Name: "Widget", Quantity: 2, UnitPrice: 50.00}},
			wantSubtotal: 100.00,
			wantDiscount: 0.00,
			wantTotal:    100.00,
		},
		{
			name:         "SAVE5 bronze 5%",
			items:        []Item{{Name: "Widget", Quantity: 1, UnitPrice: 100.00}},
			discountCode: "SAVE5",
			wantSubtotal: 100.00,
			wantDiscount: 5.00,
			wantTotal:    95.00,
		},
		{
			name:         "SAVE10 silver 10%",
			items:        []Item{{Name: "Widget", Quantity: 1, UnitPrice: 100.00}},
			discountCode: "SAVE10",
			wantSubtotal: 100.00,
			wantDiscount: 10.00,
			wantTotal:    90.00,
		},
		{
			name:         "SAVE15 gold 15%",
			items:        []Item{{Name: "Widget", Quantity: 1, UnitPrice: 100.00}},
			discountCode: "SAVE15",
			wantSubtotal: 100.00,
			wantDiscount: 15.00,
			wantTotal:    85.00,
		},
		{
			name:         "multiple items",
			items:        []Item{{Name: "Alpha", Quantity: 3, UnitPrice: 10.00}, {Name: "Beta", Quantity: 2, UnitPrice: 25.00}},
			wantSubtotal: 80.00,
			wantDiscount: 0.00,
			wantTotal:    80.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := createTestOrder(t, tt.items, tt.discountCode)

			if order.Status != "confirmed" {
				t.Errorf("status = %q, want confirmed", order.Status)
			}
			if order.Subtotal != tt.wantSubtotal {
				t.Errorf("subtotal = %.2f, want %.2f", order.Subtotal, tt.wantSubtotal)
			}
			if order.DiscountAmount != tt.wantDiscount {
				t.Errorf("discount_amount = %.2f, want %.2f", order.DiscountAmount, tt.wantDiscount)
			}
			if order.Total != tt.wantTotal {
				t.Errorf("total = %.2f, want %.2f", order.Total, tt.wantTotal)
			}
			if order.ID == "" {
				t.Error("expected non-empty order ID")
			}
			if order.CreatedAt == "" {
				t.Error("expected non-empty created_at")
			}
		})
	}
}

func TestCreateOrderErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		wantStatus int
		wantError  string
	}{
		{
			name:       "empty items",
			body:       CreateRequest{Items: []Item{}},
			wantStatus: http.StatusBadRequest,
			wantError:  "order must contain at least one item",
		},
		{
			name:       "invalid JSON",
			body:       nil, // we send raw invalid JSON below
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
		{
			name:       "unrecognised discount code",
			body:       CreateRequest{Items: []Item{{Name: "Widget", Quantity: 1, UnitPrice: 10.00}}, DiscountCode: "BOGUS"},
			wantStatus: http.StatusBadRequest,
			wantError:  "unrecognised discount code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			var body []byte

			if tt.name == "invalid JSON" {
				// Send raw invalid JSON.
				r, err := client.Post(targetURL+"/orders", "application/json", strings.NewReader("{not json"))
				if err != nil {
					t.Fatalf("POST /orders: %v", err)
				}
				defer r.Body.Close()
				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read response: %v", err)
				}
				resp = r
				body = b
			} else {
				resp, body = doPost(t, "/orders", tt.body)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d: %s", resp.StatusCode, tt.wantStatus, body)
			}
			if !strings.Contains(string(body), tt.wantError) {
				t.Errorf("body = %s, want to contain %q", body, tt.wantError)
			}
		})
	}
}

func TestGetOrder(t *testing.T) {
	created := createTestOrder(t, []Item{{Name: "Lookup", Quantity: 1, UnitPrice: 42.00}}, "")

	resp, body := doGet(t, "/orders/"+created.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	got := decodeOrder(t, body)
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Total != created.Total {
		t.Errorf("total = %.2f, want %.2f", got.Total, created.Total)
	}
}

func TestGetOrderNotFound(t *testing.T) {
	resp, body := doGet(t, "/orders/00000000-0000-0000-0000-000000000000")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "order not found") {
		t.Errorf("body = %s, want to contain 'order not found'", body)
	}
}

func TestListOrders(t *testing.T) {
	// Ensure at least one order exists.
	createTestOrder(t, []Item{{Name: "ListTest", Quantity: 1, UnitPrice: 5.00}}, "")

	resp, body := doGet(t, "/orders")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var orders []Order
	if err := json.Unmarshal(body, &orders); err != nil {
		t.Fatalf("decode orders: %v", err)
	}
	if len(orders) < 1 {
		t.Fatal("expected at least 1 order")
	}

	// Filter by status.
	resp, body = doGet(t, "/orders?status=confirmed")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for status filter, got %d: %s", resp.StatusCode, body)
	}
	var filtered []Order
	if err := json.Unmarshal(body, &filtered); err != nil {
		t.Fatalf("decode filtered orders: %v", err)
	}
	for _, o := range filtered {
		if o.Status != "confirmed" {
			t.Errorf("expected status=confirmed, got %q for order %s", o.Status, o.ID)
		}
	}
}

func TestRefundOrder(t *testing.T) {
	order := createTestOrder(t, []Item{{Name: "Refundable", Quantity: 1, UnitPrice: 30.00}}, "")

	// Refund the order.
	resp, body := doPost(t, fmt.Sprintf("/orders/%s/refund", order.ID), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	refunded := decodeOrder(t, body)
	if refunded.Status != "refunded" {
		t.Errorf("status = %q, want refunded", refunded.Status)
	}

	// Refund again — should be 409 Conflict.
	resp, body = doPost(t, fmt.Sprintf("/orders/%s/refund", order.ID), nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "order already refunded") {
		t.Errorf("body = %s, want to contain 'order already refunded'", body)
	}
}

func TestRefundNotFound(t *testing.T) {
	resp, body := doPost(t, "/orders/00000000-0000-0000-0000-000000000000/refund", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "order not found") {
		t.Errorf("body = %s, want to contain 'order not found'", body)
	}
}

func TestGitPATValid(t *testing.T) {
	// Retrieve the PAT from Secrets Manager via the AWS CLI.
	// Uses AWS_REGION and AWS_PROFILE from the environment (set via .env / justfile).
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-west-2"
	}
	args := []string{"secretsmanager", "get-secret-value",
		"--secret-id", "demo-incident-response/git-pat",
		"--region", region,
		"--query", "SecretString",
		"--output", "text",
	}
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		args = append(args, "--profile", profile)
	}
	out, err := exec.Command("aws", args...).Output()
	if err != nil {
		t.Fatalf("failed to retrieve Git PAT from Secrets Manager: %v", err)
	}

	pat := strings.TrimSpace(string(out))
	if pat == "" {
		t.Fatal("Git PAT is empty in Secrets Manager")
	}

	gitProvider := os.Getenv("GIT_PROVIDER")
	if gitProvider == "" {
		gitProvider = "github"
	}

	var url, headerKey, headerVal string
	if gitProvider == "gitlab" {
		gitlabURL := os.Getenv("GITLAB_URL")
		if gitlabURL == "" {
			gitlabURL = "https://gitlab.com"
		}
		url = gitlabURL + "/api/v4/user"
		headerKey = "PRIVATE-TOKEN"
		headerVal = pat
	} else {
		url = "https://api.github.com/user"
		headerKey = "Authorization"
		headerVal = "token " + pat
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set(headerKey, headerVal)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s API request failed: %v", gitProvider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s PAT is invalid or lacks permissions: status %d", gitProvider, resp.StatusCode)
	}
}
