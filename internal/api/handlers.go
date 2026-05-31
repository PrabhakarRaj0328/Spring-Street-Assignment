package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"spring-street-backend/internal/models"
)

type Server struct {
	DB *sql.DB
}

func NewServer(db *sql.DB) *Server {
	return &Server{DB: db}
}

func (s *Server) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/fund/{id}", s.GetFundOverview).Methods("GET")
	r.HandleFunc("/api/fund/{id}/holdings", s.GetFundHoldings).Methods("GET")
	r.HandleFunc("/api/fund/{id}/exposure/sector", s.GetSectorExposure).Methods("GET")
	r.HandleFunc("/api/fund/{id}/exposure/country", s.GetCountryExposure).Methods("GET")
	r.HandleFunc("/api/fund/{id}/exposure/marketcap", s.GetMarketCapExposure).Methods("GET")
	r.HandleFunc("/api/fund/{id}/performance", s.GetPerformance).Methods("GET")
	r.HandleFunc("/api/fund/{id}/nav-history", s.GetNAVHistory).Methods("GET")
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{"error": message})
}

func (s *Server) GetFundOverview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var fund models.Fund
	err := s.DB.QueryRow(`
		SELECT id, name, ticker, TO_CHAR(inception_date, 'YYYY-MM-DD'), benchmark, expense_ratio 
		FROM funds WHERE id = $1`, id).
		Scan(&fund.ID, &fund.Name, &fund.Ticker, &fund.InceptionDate, &fund.Benchmark, &fund.ExpenseRatio)
	
	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Fund not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, fund)
}

func (s *Server) GetFundHoldings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rows, err := s.DB.Query(`
		SELECT stock_name, ticker, weight_percent, market_cap_category 
		FROM holdings WHERE fund_id = $1 ORDER BY weight_percent DESC`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var holdings = make([]models.Holding, 0)
	for rows.Next() {
		var h models.Holding
		if err := rows.Scan(&h.StockName, &h.Ticker, &h.WeightPercent, &h.MarketCapCategory); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		holdings = append(holdings, h)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, holdings)
}

func (s *Server) GetSectorExposure(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rows, err := s.DB.Query(`
		SELECT sector_name, allocation_percent 
		FROM sector_exposures WHERE fund_id = $1 ORDER BY allocation_percent DESC`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var exposures = make([]models.SectorExposure, 0)
	for rows.Next() {
		var e models.SectorExposure
		if err := rows.Scan(&e.SectorName, &e.AllocationPercent); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		exposures = append(exposures, e)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, exposures)
}

func (s *Server) GetCountryExposure(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rows, err := s.DB.Query(`
		SELECT country, region, allocation_percent 
		FROM country_exposures WHERE fund_id = $1 ORDER BY allocation_percent DESC`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var exposures = make([]models.CountryExposure, 0)
	for rows.Next() {
		var e models.CountryExposure
		if err := rows.Scan(&e.Country, &e.Region, &e.AllocationPercent); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		exposures = append(exposures, e)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, exposures)
}

func (s *Server) GetMarketCapExposure(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rows, err := s.DB.Query(`
		SELECT category, allocation_percent 
		FROM market_cap_breakdowns WHERE fund_id = $1 ORDER BY allocation_percent DESC`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var exposures = make([]models.MarketCapBreakdown, 0)
	for rows.Next() {
		var e models.MarketCapBreakdown
		if err := rows.Scan(&e.Category, &e.AllocationPercent); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		exposures = append(exposures, e)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, exposures)
}

func (s *Server) GetPerformance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rows, err := s.DB.Query(`
		SELECT period, return_percent 
		FROM performances WHERE fund_id = $1`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var perfs = make([]models.Performance, 0)
	for rows.Next() {
		var p models.Performance
		if err := rows.Scan(&p.Period, &p.ReturnPercent); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		perfs = append(perfs, p)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, perfs)
}

func (s *Server) GetNAVHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rows, err := s.DB.Query(`
		SELECT TO_CHAR(date, 'YYYY-MM-DD'), price, nav 
		FROM daily_navs WHERE fund_id = $1 ORDER BY date ASC`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var navs = make([]models.DailyNAV, 0)
	for rows.Next() {
		var n models.DailyNAV
		if err := rows.Scan(&n.Date, &n.Price, &n.NAV); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		navs = append(navs, n)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, navs)
}
