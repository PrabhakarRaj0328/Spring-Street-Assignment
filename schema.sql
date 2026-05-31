CREATE TABLE IF NOT EXISTS funds (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    ticker VARCHAR(50) NOT NULL UNIQUE,
    inception_date DATE,
    benchmark VARCHAR(255),
    expense_ratio NUMERIC(5, 4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS daily_navs (
    id SERIAL PRIMARY KEY,
    fund_id INTEGER REFERENCES funds(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    price NUMERIC(15, 4) NOT NULL,
    nav NUMERIC(15, 4),
    UNIQUE (fund_id, date)
);

CREATE TABLE IF NOT EXISTS holdings (
    id SERIAL PRIMARY KEY,
    fund_id INTEGER REFERENCES funds(id) ON DELETE CASCADE,
    stock_name VARCHAR(255) NOT NULL,
    ticker VARCHAR(50),
    weight_percent NUMERIC(5, 2) NOT NULL,
    market_cap_category VARCHAR(50),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sector_exposures (
    id SERIAL PRIMARY KEY,
    fund_id INTEGER REFERENCES funds(id) ON DELETE CASCADE,
    sector_name VARCHAR(100) NOT NULL,
    allocation_percent NUMERIC(5, 2) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (fund_id, sector_name)
);

CREATE TABLE IF NOT EXISTS country_exposures (
    id SERIAL PRIMARY KEY,
    fund_id INTEGER REFERENCES funds(id) ON DELETE CASCADE,
    country VARCHAR(100) NOT NULL,
    region VARCHAR(100),
    allocation_percent NUMERIC(5, 2) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (fund_id, country)
);

CREATE TABLE IF NOT EXISTS market_cap_breakdowns (
    id SERIAL PRIMARY KEY,
    fund_id INTEGER REFERENCES funds(id) ON DELETE CASCADE,
    category VARCHAR(50) NOT NULL, -- e.g., 'Large', 'Mid', 'Small'
    allocation_percent NUMERIC(5, 2) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (fund_id, category)
);

CREATE TABLE IF NOT EXISTS performances (
    id SERIAL PRIMARY KEY,
    fund_id INTEGER REFERENCES funds(id) ON DELETE CASCADE,
    period VARCHAR(50) NOT NULL, -- e.g., '1M', '3M', '6M', '1Y', '3Y', 'Inception'
    return_percent NUMERIC(10, 4) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (fund_id, period)
);

-- Insert Prisma Fund metadata mock
INSERT INTO funds (name, ticker, inception_date, benchmark, expense_ratio) 
VALUES ('Prisma Global Growth Fund', 'PRISMA', '2023-01-01', 'MSCI World Index', 0.0075)
ON CONFLICT (ticker) DO NOTHING;
