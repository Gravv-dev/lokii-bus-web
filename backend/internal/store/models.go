package store

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID          string    `json:"id"`
	CategoryID  *string   `json:"category_id,omitempty"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url"`
	Price       float64   `json:"price"`
	Active      bool      `json:"active"`
	Inventory   int       `json:"inventory"`
	CreatedAt   time.Time `json:"created_at"`
}

type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type CartItem struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	ImageURL  string  `json:"image_url"`
	UnitPrice float64 `json:"unit_price"`
	Quantity  int     `json:"quantity"`
	Subtotal  float64 `json:"subtotal"`
}

type Cart struct {
	Items []CartItem `json:"items"`
	Total float64    `json:"total"`
}

type OrderItem struct {
	ProductID *string `json:"product_id,omitempty"`
	Name      string  `json:"name"`
	UnitPrice float64 `json:"unit_price"`
	Quantity  int     `json:"quantity"`
}

type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id,omitempty"`
	Status    string      `json:"status"`
	Total     float64     `json:"total"`
	Items     []OrderItem `json:"items"`
	CreatedAt time.Time   `json:"created_at"`
}

type ProductInput struct {
	CategoryID  *string `json:"category_id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	ImageURL    string  `json:"image_url"`
	Price       float64 `json:"price"`
	Active      *bool   `json:"active"`
	Inventory   int     `json:"inventory"`
}

type CategoryInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}
