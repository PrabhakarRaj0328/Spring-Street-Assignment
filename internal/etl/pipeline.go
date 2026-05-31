package etl

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Pipeline struct {
	DB *sql.DB
}

func NewPipeline(db *sql.DB) *Pipeline {
	return &Pipeline{DB: db}
}

// YahooFinanceChartResponse maps the JSON structure of the chart endpoint
type YahooFinanceChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				Symbol             string  `json:"symbol"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

func (p *Pipeline) RunDaily() error {
	log.Println("Starting daily ETL pipeline for Prisma fund...")

	// 1. Fetch Prisma fund ID
	var fundID int
	err := p.DB.QueryRow("SELECT id FROM funds WHERE ticker = 'PRISMA'").Scan(&fundID)
	if err != nil {
		return fmt.Errorf("could not find PRISMA fund: %v", err)
	}

	// 2. Fetch market data from Yahoo Finance (mocking Prisma using a global ETF like VT as proxy for prices)
	ticker := "VT"
	yfResp, fetchErr := p.fetchWithRetry(ticker, 3)

	if fetchErr != nil {
		log.Printf("WARNING: Yahoo Finance fetch failed after retries: %v. Skipping NAV ingestion, continuing with mock data.", fetchErr)
	}
	
	// 3. Process and Load NAVs into Database
	tx, err := p.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Insert historical daily NAVs (only if Yahoo Finance fetch succeeded)
	if fetchErr == nil && len(yfResp.Chart.Result) > 0 {
		result := yfResp.Chart.Result[0]
		for i, ts := range result.Timestamp {
			date := time.Unix(ts, 0).Format("2006-01-02")

			if i >= len(result.Indicators.Quote[0].Close) {
				continue
			}

			price := result.Indicators.Quote[0].Close[i]
			if price == 0 { // Skip 0 or nil values
				continue
			}

			_, err = tx.Exec(`
				INSERT INTO daily_navs (fund_id, date, price, nav) 
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (fund_id, date) DO UPDATE SET price = EXCLUDED.price, nav = EXCLUDED.nav`,
				fundID, date, price, price*10000) // Mock NAV calculation
			if err != nil {
				return fmt.Errorf("failed to insert NAV for %s: %v", date, err)
			}
		}
		log.Printf("Ingested %d NAV records from Yahoo Finance", len(result.Timestamp))
	}

	// 4. Update Holdings (Mocking typical global fund distribution)
	err = p.updateMockHoldings(tx, fundID)
	if err != nil {
		return err
	}

	// 5. Update Exposures
	err = p.updateMockExposures(tx, fundID)
	if err != nil {
		return err
	}

	// 6. Update Performances
	err = p.updateMockPerformances(tx, fundID)
	if err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Println("ETL pipeline completed successfully.")
	return nil
}

// Below functions populate the database with structural mock data to fulfill the API requirements.
// In a real-world scenario, these would fetch from the `quoteSummary` endpoint of Yahoo Finance or a premium data provider.

func (p *Pipeline) updateMockHoldings(tx *sql.Tx, fundID int) error {
	// Prisma Global Growth is an ETF-based portfolio, not individual stocks.
	// Holdings reflect the underlying ETF basket structure.
	holdings := []struct {
		Name   string
		Ticker string
		Weight float64
		Cap    string
	}{
		{"Vanguard Total World Stock ETF", "VT", 25.0, "Large"},
		{"Vanguard Total Stock Market ETF", "VTI", 20.0, "Large"},
		{"Vanguard FTSE Developed Markets ETF", "VEA", 18.0, "Large"},
		{"Vanguard FTSE Emerging Markets ETF", "VWO", 15.0, "Large"},
		{"iShares Core MSCI EAFE ETF", "IEFA", 12.0, "Large"},
		{"iShares MSCI Emerging Markets ETF", "EEM", 10.0, "Mid"},
	}

	_, err := tx.Exec("DELETE FROM holdings WHERE fund_id = $1", fundID)
	if err != nil {
		return err
	}

	for _, h := range holdings {
		_, err = tx.Exec(`
			INSERT INTO holdings (fund_id, stock_name, ticker, weight_percent, market_cap_category)
			VALUES ($1, $2, $3, $4, $5)`,
			fundID, h.Name, h.Ticker, h.Weight, h.Cap)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Pipeline) updateMockExposures(tx *sql.Tx, fundID int) error {
	sectors := map[string]float64{
		"Information Technology": 22.5,
		"Financials":             14.2,
		"Health Care":            12.8,
		"Consumer Discretionary": 10.5,
		"Industrials":            9.0,
		"Communication Services": 7.5,
		"Consumer Staples":       6.8,
		"Energy":                 5.2,
		"Materials":              4.0,
		"Utilities":              3.5,
		"Real Estate":            4.0,
	}

	for sector, weight := range sectors {
		_, err := tx.Exec(`
			INSERT INTO sector_exposures (fund_id, sector_name, allocation_percent)
			VALUES ($1, $2, $3)
			ON CONFLICT (fund_id, sector_name) DO UPDATE SET allocation_percent = EXCLUDED.allocation_percent, updated_at = CURRENT_TIMESTAMP`,
			fundID, sector, weight)
		if err != nil {
			return err
		}
	}

	// Geographic exposure matching the live Prisma Global Growth product page:
	// North America 40%, Asia-Pacific 30%, South America 15%, Europe 15%
	countries := map[string]struct {
		Region string
		Weight float64
	}{
		"United States":  {"North America", 35.0},
		"Canada":         {"North America", 5.0},
		"Japan":          {"Asia-Pacific", 10.0},
		"China":          {"Asia-Pacific", 8.0},
		"India":          {"Asia-Pacific", 5.0},
		"Australia":      {"Asia-Pacific", 4.0},
		"South Korea":    {"Asia-Pacific", 3.0},
		"Brazil":         {"South America", 8.0},
		"Chile":          {"South America", 4.0},
		"Colombia":       {"South America", 3.0},
		"United Kingdom": {"Europe", 5.0},
		"Germany":        {"Europe", 4.0},
		"France":         {"Europe", 3.5},
		"Switzerland":    {"Europe", 2.5},
	}

	for country, data := range countries {
		_, err := tx.Exec(`
			INSERT INTO country_exposures (fund_id, country, region, allocation_percent)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (fund_id, country) DO UPDATE SET allocation_percent = EXCLUDED.allocation_percent, updated_at = CURRENT_TIMESTAMP`,
			fundID, country, data.Region, data.Weight)
		if err != nil {
			return err
		}
	}

	caps := map[string]float64{
		"Large": 70.0,
		"Mid":   18.0,
		"Small": 12.0,
	}

	for capCategory, weight := range caps {
		_, err := tx.Exec(`
			INSERT INTO market_cap_breakdowns (fund_id, category, allocation_percent)
			VALUES ($1, $2, $3)
			ON CONFLICT (fund_id, category) DO UPDATE SET allocation_percent = EXCLUDED.allocation_percent, updated_at = CURRENT_TIMESTAMP`,
			fundID, capCategory, weight)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pipeline) updateMockPerformances(tx *sql.Tx, fundID int) error {
	// Performance data inspired by the live Prisma Global Growth product page.
	// Includes YTD and CAGR which are prominently displayed on the factsheet.
	perfs := map[string]float64{
		"1M":        2.5,
		"3M":        5.8,
		"6M":        12.4,
		"YTD":       10.5,
		"1Y":        22.1,
		"3Y":        35.6,
		"CAGR":      16.6,
		"Inception": 45.0,
	}

	for period, returnPct := range perfs {
		_, err := tx.Exec(`
			INSERT INTO performances (fund_id, period, return_percent)
			VALUES ($1, $2, $3)
			ON CONFLICT (fund_id, period) DO UPDATE SET return_percent = EXCLUDED.return_percent, updated_at = CURRENT_TIMESTAMP`,
			fundID, period, returnPct)
		if err != nil {
			return err
		}
	}
	return nil
}

// fetchWithRetry fetches data from Yahoo Finance with exponential backoff retry logic.
// Yahoo Finance requires cookie + crumb authentication for its unofficial API.
func (p *Pipeline) fetchWithRetry(ticker string, maxRetries int) (*YahooFinanceChartResponse, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			log.Printf("Retry %d/%d for Yahoo Finance (waiting %v)...", attempt+1, maxRetries, backoff)
			time.Sleep(backoff)
		}

		yfResp, err := p.fetchWithAuth(ticker)
		if err != nil {
			lastErr = err
			continue
		}
		return yfResp, nil
	}

	return nil, fmt.Errorf("all %d retries exhausted: %v", maxRetries, lastErr)
}

// fetchWithAuth performs the Yahoo Finance cookie+crumb authentication flow:
// 1. Visit finance.yahoo.com to obtain session cookies
// 2. Fetch crumb token from /v1/test/getcrumb
// 3. Use cookie+crumb to call the chart API
func (p *Pipeline) fetchWithAuth(ticker string) (*YahooFinanceChartResponse, error) {
	// Create a cookie jar to persist session cookies across requests
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
	}

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

	// Step 1: Visit Yahoo Finance to get session cookies
	log.Println("Fetching Yahoo Finance session cookies...")
	cookieReq, err := http.NewRequest("GET", "https://finance.yahoo.com/quote/VT/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie request: %v", err)
	}
	cookieReq.Header.Set("User-Agent", userAgent)
	cookieReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	cookieReq.Header.Set("Accept-Language", "en-US,en;q=0.5")

	cookieResp, err := client.Do(cookieReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session cookies: %v", err)
	}
	cookieResp.Body.Close()

	// Verify we got cookies
	yahooURL, _ := url.Parse("https://finance.yahoo.com")
	cookies := jar.Cookies(yahooURL)
	if len(cookies) == 0 {
		return nil, fmt.Errorf("no session cookies received from Yahoo Finance")
	}
	log.Printf("Received %d session cookies", len(cookies))

	// Step 2: Fetch crumb token
	log.Println("Fetching crumb token...")
	crumbReq, err := http.NewRequest("GET", "https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create crumb request: %v", err)
	}
	crumbReq.Header.Set("User-Agent", userAgent)
	crumbReq.Header.Set("Accept", "*/*")
	crumbReq.Header.Set("Referer", "https://finance.yahoo.com")

	crumbResp, err := client.Do(crumbReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch crumb: %v", err)
	}
	defer crumbResp.Body.Close()

	if crumbResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crumb endpoint returned HTTP %d", crumbResp.StatusCode)
	}

	crumbBytes, err := io.ReadAll(crumbResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read crumb response: %v", err)
	}
	crumb := strings.TrimSpace(string(crumbBytes))
	if crumb == "" {
		return nil, fmt.Errorf("received empty crumb token")
	}
	log.Printf("Obtained crumb token: %s", crumb)

	// Step 3: Fetch chart data with cookie + crumb
	chartURL := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?range=1mo&interval=1d&crumb=%s",
		ticker, url.QueryEscape(crumb),
	)

	chartReq, err := http.NewRequest("GET", chartURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chart request: %v", err)
	}
	chartReq.Header.Set("User-Agent", userAgent)
	chartReq.Header.Set("Accept", "application/json")
	chartReq.Header.Set("Referer", "https://finance.yahoo.com")

	chartResp, err := client.Do(chartReq)
	if err != nil {
		return nil, fmt.Errorf("chart request failed: %v", err)
	}
	defer chartResp.Body.Close()

	if chartResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Yahoo Finance chart API returned HTTP %d", chartResp.StatusCode)
	}

	body, err := io.ReadAll(chartResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read chart response: %v", err)
	}

	var yfResp YahooFinanceChartResponse
	if err := json.Unmarshal(body, &yfResp); err != nil {
		return nil, fmt.Errorf("failed to parse Yahoo Finance data: %v", err)
	}

	if yfResp.Chart.Error != nil {
		return nil, fmt.Errorf("Yahoo Finance API error: %v", yfResp.Chart.Error)
	}

	if len(yfResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data returned from Yahoo Finance")
	}

	log.Printf("Successfully fetched %d data points from Yahoo Finance", len(yfResp.Chart.Result[0].Timestamp))
	return &yfResp, nil
}
