package order

import "encoding/json"

// Order definition
type Order struct {
	Id              string `json:"id"`
	ProductCategory string `json:"ProductCategory"`
	ProductID       string `json:"productId"`
	CustomerID      string `json:"customerId"`
	Status          string `json:"status"`
}

// Serialize transforms the Order instance into a []byte
func (e *Order) Serialize() ([]byte, error) {
	return json.Marshal(e)
}

// Convert Order struct into a map[string]string
func (e *Order) ToMap() map[string]string {
	return map[string]string{
		"Id":              e.Id,
		"ProductCategory": e.ProductCategory,
		"ProductID":       e.ProductID,
		"CustomerID":      e.CustomerID,
		"Status":          e.Status,
	}
}
