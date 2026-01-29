package models

type Transaction struct {
	ID                string  `json:"id" csv:"id"`
	Timestamp         string  `json:"timestamp" csv:"timestamp"`
	Title             string  `json:"title" csv:"title"`
	Icon              string  `json:"icon" csv:"icon"`
	Avatar            string  `json:"avatar.asset" csv:"avatar.asset"`
	Subtitle          string  `json:"subtitle" csv:"subtitle"`
	AmountCurrency    string  `json:"amount.currency" csv:"amount.currency"`
	AmountValue       float64 `json:"amount.value" csv:"amount.value"`
	AmountFraction    int     `json:"amount.fractionDigits" csv:"amount.fractionDigits"`
	Status            string  `json:"status" csv:"status"`
	ActionType        string  `json:"action.type" csv:"action.type"`
	ActionPayload     string  `json:"action.payload" csv:"action.payload"`
	CashAccountNumber string  `json:"cashAccountNumber" csv:"cashAccountNumber"`
	Hidden            bool    `json:"hidden" csv:"hidden"`
	Deleted           bool    `json:"deleted" csv:"deleted"`
	
	// Details (when extract_details is true)
	Actions           string  `json:"Actions,omitempty" csv:"Actions"`
	DividendPerShare  string  `json:"Dividende par action,omitempty" csv:"Dividende par action"`
	Taxes             string  `json:"Taxes,omitempty" csv:"Taxes"`
	Total             string  `json:"Total,omitempty" csv:"Total"`
	Shares            string  `json:"Titres,omitempty" csv:"Titres"`
	SharePrice        string  `json:"Cours du titre,omitempty" csv:"Cours du titre"`
	Fees              string  `json:"Frais,omitempty" csv:"Frais"`
	Amount            string  `json:"Montant,omitempty" csv:"Montant"`
}

type ProfileCash struct {
	Currency       string  `json:"currency" csv:"currency"`
	Value          float64 `json:"value" csv:"value"`
	FractionDigits int     `json:"fractionDigits" csv:"fractionDigits"`
}
