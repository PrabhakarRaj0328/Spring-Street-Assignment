# Spring Street Prisma Backend

A production-quality backend system powering the financial product factsheet page for the Prisma global equity investment product.

## Architecture & Data Flow

1. **Yahoo Finance → ETL Pipeline**: A Go-based ETL job runs daily (using a Go time ticker or a cron scheduler). It queries the Yahoo Finance REST endpoints for the daily price/NAV history. For the complex portfolio distributions (holdings, sector exposures, country exposures, market cap breakdowns) and performance metrics, it computes and processes the values representing the Prisma basket (in this implementation, mocked with structural data simulating a global equity fund, though in production it would aggregate from the underlying ETF assets via the YF API).
2. **ETL Pipeline → PostgreSQL**: The pipeline sanitizes and transforms the JSON data into structured SQL inserts. It uses `ON CONFLICT` constraints to gracefully perform daily upserts, maintaining data freshness without duplication. It writes to normalized, highly typed tables.
3. **PostgreSQL → REST API**: The Go REST API connects to the database and serves requests asynchronously using `gorilla/mux`. The API endpoints fetch strictly read-only data, with queries optimized by indexed primary keys and foreign keys.
4. **REST API → Client (React Frontend)**: The API serves JSON structures representing the fund metadata, holdings, exposures, and historical performance.

## Design Choices & Trade-offs

- **PostgreSQL**: Chosen for ACID compliance, strong relational integrity (foreign keys ensure child data is orphaned if a fund is deleted), and fast queries for analytical reads.
- **Go (`net/http` + `gorilla/mux`)**: Chosen for performance, simplicity, and low memory footprint. We avoided massive web frameworks like `gin` or ORMs like `gorm` to keep the dependency tree light, reducing the attack surface and making compilation lightning fast. Using standard `database/sql` gives us fine-grained control over connection pooling and SQL queries.
- **Two Binaries**: We separated the project into two applications (`cmd/api` and `cmd/etl`).
  - *Trade-off*: Slightly more complex deployment than a monolith.
  - *Benefit*: The API scales horizontally (stateless), while the ETL pipeline runs as a single scheduler instance, preventing race conditions or duplicated API calls to Yahoo Finance.
- **Mocked Portfolio Data**: The unofficial Yahoo Finance API does not easily expose nested fund breakdowns cleanly without scraping or using undocumented endpoints. For robustness and to demonstrate schema design, the pipeline mocks the internal calculation of exposures and holdings. In a real-world scenario, you would license a provider like Bloomberg, Morningstar, or use the official Yahoo Finance paid API.

## Scaling

- **If more funds are added**: The schema uses a `funds` table with a `fund_id` foreign key in all child tables. Adding a new fund is as simple as inserting a new row into the `funds` table. The ETL pipeline can easily be refactored to query `SELECT ticker FROM funds` and loop through all tickers asynchronously using Go routines and `sync.WaitGroup` to fetch data concurrently.
- **Handling High API Traffic**: The REST API is stateless. It can be horizontally scaled using Kubernetes pods or AWS ECS behind a load balancer. Read replicas for PostgreSQL can be added, pointing the API to read from the replica while the ETL writes to the primary DB.
- **Caching**: We could introduce Redis or an in-memory cache (like `ristretto` or Go `sync.Map`) for the API handlers since factsheet data only changes once daily.

## Enhancements & Robustness

- **Yahoo Finance API Auth**: Yahoo Finance's unofficial chart API actively blocks automated requests (HTTP 429). The ETL pipeline implements a robust workaround using a cookie jar, session cookies, and dynamic `crumb` token generation, wrapped in an exponential backoff retry loop. If Yahoo entirely blocks the IP, it gracefully falls back to skipping NAV history while keeping all other portfolio data updated.
- **Frontend-Ready**: Includes custom CORS middleware to allow cross-origin requests from the React frontend, and ensures empty DB queries return `[]` instead of `null` to prevent frontend crashes.
- **Accurate Product Modeling**: The mock data in the ETL accurately reflects the real-world Prisma Global Growth factsheet, utilizing institutional ETFs (VT, VTI, VEA, etc.) and matching the exact 40% NA / 30% APAC / 15% EU / 15% SA regional split.

## Setup Instructions (Fully Dockerized)

The entire application (Database, API, and ETL worker) is containerized for zero-friction deployment.

### Prerequisites
- Docker & Docker Compose

### 1. Start the Stack
From the root of the project, run:
```bash
docker-compose up --build -d
```
This will:
1. Spin up the PostgreSQL database and initialize the schema.
2. Build and start the ETL worker to fetch data.
3. Build and start the API server on port 8080.

### 2. Verify Services
Check that all three containers (`spring-street-db`, `spring-street-api`, `spring-street-etl`) are running:
```bash
docker-compose ps
```

### 3. Test Endpoints
The API is now running locally on port 8080:
- **Fund Overview**: `curl http://localhost:8080/api/fund/1`
- **Holdings**: `curl http://localhost:8080/api/fund/1/holdings`
- **Sector Exposure**: `curl http://localhost:8080/api/fund/1/exposure/sector`
- **Country Exposure**: `curl http://localhost:8080/api/fund/1/exposure/country`
- **Market Cap**: `curl http://localhost:8080/api/fund/1/exposure/marketcap`
- **Performance**: `curl http://localhost:8080/api/fund/1/performance`
- **NAV History**: `curl http://localhost:8080/api/fund/1/nav-history`
