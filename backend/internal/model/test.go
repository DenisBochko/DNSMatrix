package model

type TestTable struct {
	ID   int    `json:"id" db:"id"`
	Data string `json:"data" db:"data"`
}
