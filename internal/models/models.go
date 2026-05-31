package models

type Fund struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Ticker        string  `json:"ticker"`
	InceptionDate string  `json:"inception_date"`
	Benchmark     string  `json:"benchmark"`
	ExpenseRatio  float64 `json:"expense_ratio"`
}

type DailyNAV struct {
	Date  string  `json:"date"`
	Price float64 `json:"price"`
	NAV   float64 `json:"nav"`
}

type Holding struct {
	StockName         string  `json:"stock_name"`
	Ticker            string  `json:"ticker"`
	WeightPercent     float64 `json:"weight_percent"`
	MarketCapCategory string  `json:"market_cap_category"`
}

type SectorExposure struct {
	SectorName        string  `json:"sector_name"`
	AllocationPercent float64 `json:"allocation_percent"`
}

type CountryExposure struct {
	Country           string  `json:"country"`
	Region            string  `json:"region"`
	AllocationPercent float64 `json:"allocation_percent"`
}

type MarketCapBreakdown struct {
	Category          string  `json:"category"`
	AllocationPercent float64 `json:"allocation_percent"`
}

type Performance struct {
	Period        string  `json:"period"`
	ReturnPercent float64 `json:"return_percent"`
}
