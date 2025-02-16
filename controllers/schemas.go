package controllers

type TokenResponse struct {
	SignedToken string `json:"token"`
}

type SendToPayload struct {
	ToUser string  `json:"toUser" binding:"required"`
	Amount float32 `json:"amount" binding:"required"`
}

type InventorySchema struct {
	Type     string `json:"type"`
	Quantity uint64 `json:"quantity"`
}

type ReceivedSchema struct {
	FromUser string  `gorm:"column:from_user" json:"fromUser"`
	Amount   float32 `json:"amount"`
}

type SentSchema struct {
	ToUser string  `gorm:"column:to_user" json:"toUser"`
	Amount float32 `json:"amount"`
}

type HistorySchema struct {
	Received []ReceivedSchema `json:"received"`
	Sent     []SentSchema     `json:"sent"`
}
type InfoSchema struct {
	Coins       float32           `json:"coins"`
	Inventory   []InventorySchema `json:"inventory"`
	CoinHistory HistorySchema     `json:"coinHistory"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}
