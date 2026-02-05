package binance

import (
	"valhafin/internal/config"
)

// TODO: Implement Binance API client
// Will use official Binance API with API key and secret

type Client struct {
	config *config.Config
	apiKey string
	secret string
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		// TODO: Load from config
	}
}

// FetchTransactions retrieves transaction history from Binance
func (c *Client) FetchTransactions() ([]interface{}, error) {
	// TODO: Implement using Binance REST API
	// GET /api/v3/myTrades
	return nil, nil
}

// FetchDeposits retrieves deposit history
func (c *Client) FetchDeposits() ([]interface{}, error) {
	// TODO: Implement
	// GET /sapi/v1/capital/deposit/hisrec
	return nil, nil
}

// FetchWithdrawals retrieves withdrawal history
func (c *Client) FetchWithdrawals() ([]interface{}, error) {
	// TODO: Implement
	// GET /sapi/v1/capital/withdraw/history
	return nil, nil
}
