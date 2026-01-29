# Valhafin 🔥⚔️

**Your Financial Valhalla**

*Where wealth warriors ascend*

A high-performance Go application to scrape and aggregate financial data from multiple sources (Trade Republic, Binance, Bourse Direct, etc.).

Named after Valhalla, the hall of slain heroes in Norse mythology - your ultimate destination for financial glory.

## Supported Sources

- ✅ Trade Republic
- 🚧 Binance (coming soon)
- 🚧 Bourse Direct (coming soon)

## Installation

```bash
go mod download
```

## Configuration

Copy `config.yaml.example` and edit with your credentials:

```yaml
secret:
  phone_number: "+33XXXXXXXXX"
  pin: "XXXX"

general:
  output_format: "csv"  # json or csv
  output_folder: "out"
  extract_details: true
```

## Usage

```bash
go run main.go
```

## Build

```bash
go build -o valhafin
./valhafin
```

## Project Structure

```
valhafin/
├── main.go                 # Entry point
├── config/                 # Configuration management
├── scrapers/              # Scraper implementations
│   ├── traderepublic/    # Trade Republic scraper
│   ├── binance/          # Binance API client
│   └── boursedirect/     # Bourse Direct scraper
├── models/               # Data models
├── utils/                # Utilities (CSV, JSON export)
└── out/                  # Output directory
```

## License

MIT
