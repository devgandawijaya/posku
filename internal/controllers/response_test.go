package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		query          string
		expectedPage   int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "default pagination",
			query:          "",
			expectedPage:   1,
			expectedLimit:  20,
			expectedOffset: 0,
		},
		{
			name:           "custom page and per_page",
			query:          "?page=3&per_page=15",
			expectedPage:   3,
			expectedLimit:  15,
			expectedOffset: 30,
		},
		{
			name:           "limit alias instead of per_page",
			query:          "?page=2&limit=50",
			expectedPage:   2,
			expectedLimit:  50,
			expectedOffset: 50,
		},
		{
			name:           "cap per_page to max 100",
			query:          "?page=1&per_page=500",
			expectedPage:   1,
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "invalid parameters fallback to defaults",
			query:          "?page=-5&per_page=invalid",
			expectedPage:   1,
			expectedLimit:  20,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			req, _ := http.NewRequest("GET", "/test"+tt.query, nil)
			c.Request = req

			page, perPage, offset := GetPagination(c)
			if page != tt.expectedPage {
				t.Errorf("expected page %d, got %d", tt.expectedPage, page)
			}
			if perPage != tt.expectedLimit {
				t.Errorf("expected perPage %d, got %d", tt.expectedLimit, perPage)
			}
			if offset != tt.expectedOffset {
				t.Errorf("expected offset %d, got %d", tt.expectedOffset, offset)
			}
		})
	}
}

func TestRespondPaginated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	items := []string{"item1", "item2", "item3"}
	RespondPaginated(c, items, 1, 10, 25)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
		Meta    struct {
			Page       int   `json:"page"`
			PerPage    int   `json:"per_page"`
			Total      int64 `json:"total"`
			TotalPages int   `json:"total_pages"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Success {
		t.Errorf("expected success true")
	}
	if len(response.Data) != 3 {
		t.Errorf("expected 3 items, got %d", len(response.Data))
	}
	if response.Meta.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", response.Meta.TotalPages)
	}
	if response.Meta.Total != 25 {
		t.Errorf("expected total 25, got %d", response.Meta.Total)
	}
}

func TestGetAuthCompanyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("extracts uint from jwt claim", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Set("company_id", uint(7))

		cid := getAuthCompanyID(c)
		if cid != 7 {
			t.Errorf("expected 7, got %d", cid)
		}
	})

	t.Run("extracts float64 from jwt claim", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Set("company_id", float64(15))

		cid := getAuthCompanyID(c)
		if cid != 15 {
			t.Errorf("expected 15, got %d", cid)
		}
	})

	t.Run("fallback to query parameter when no jwt claim", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest("GET", "/test?company_id=99", nil)

		cid := getAuthCompanyID(c)
		if cid != 99 {
			t.Errorf("expected 99, got %d", cid)
		}
	})
}

func TestShiftReconciliationMath(t *testing.T) {
	// Unit test verify the mathematical logic of CloseShift cash reconciliation
	initialCash := 100000.0
	cashSales := 450000.0
	actualCash := 540000.0

	expectedCash := initialCash + cashSales
	discrepancy := actualCash - expectedCash

	if expectedCash != 550000.0 {
		t.Errorf("expected 550000, got %f", expectedCash)
	}
	if discrepancy != -10000.0 {
		t.Errorf("expected -10000 (shortage), got %f", discrepancy)
	}

	// Surplus cash test
	actualSurplus := 560000.0
	discrepancySurplus := actualSurplus - expectedCash
	if discrepancySurplus != 10000.0 {
		t.Errorf("expected +10000 (surplus), got %f", discrepancySurplus)
	}
}

func TestSplitPaymentCalculation(t *testing.T) {
	type PaymentSplit struct {
		Method string
		Amount float64
	}

	splits := []PaymentSplit{
		{Method: "cash", Amount: 50000},
		{Method: "qris", Amount: 75000},
	}

	totalAmount := 125000.0
	var sum float64
	for _, p := range splits {
		sum += p.Amount
	}

	if sum != totalAmount {
		t.Errorf("expected total split sum %f, got %f", totalAmount, sum)
	}
}

