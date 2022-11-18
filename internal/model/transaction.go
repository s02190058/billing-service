package model

import "time"

type Transaction struct {
	ID      int       `json:"id"`
	UserID  int       `json:"user_id"`
	Amount  int       `json:"amount"`
	Message string    `json:"message"`
	Created time.Time `json:"created"`
}
