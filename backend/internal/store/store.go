package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrInsufficientStock = errors.New("insufficient stock")
var ErrEmptyCart = errors.New("cart is empty")
var ErrInvalidTransition = errors.New("invalid order status transition")
var ErrQuantityLimit = errors.New("cart quantity exceeds limit")

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Health(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) BootstrapAdmin(ctx context.Context, email, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (email, name, password_hash, role)
		VALUES ($1, 'Administrator', $2, 'admin')
		ON CONFLICT (email) DO NOTHING`, email, string(hash))
	return err
}

func (s *Store) Register(ctx context.Context, email, name, password string) (User, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}
	var user User
	err = s.pool.QueryRow(ctx, `
		INSERT INTO users (email, name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id::text, email, name, role, created_at`, email, name, string(hash)).
		Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.CreatedAt)
	if err != nil && strings.Contains(err.Error(), "users_email_key") {
		return User{}, ErrConflict
	}
	return user, err
}

func (s *Store) Login(ctx context.Context, email, password string) (User, error) {
	var user User
	var passwordHash string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, email, name, role, created_at, password_hash
		FROM users WHERE email = $1`, email).
		Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.CreatedAt, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	if !passwordMatches([]byte(passwordHash), []byte(password)) {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *Store) CreateSession(ctx context.Context, userID, token, csrf string, expires time.Time) error {
	tokenHash := sha256.Sum256([]byte(token))
	csrfHash := sha256.Sum256([]byte(csrf))
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, csrf_hash, expires_at)
		VALUES ($1, $2, $3, $4)`, userID, tokenHash[:], csrfHash[:], expires)
	return err
}

func (s *Store) SessionUser(ctx context.Context, token string) (User, error) {
	tokenHash := sha256.Sum256([]byte(token))
	var user User
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.name, u.role, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash[:]).
		Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (s *Store) ValidateCSRF(ctx context.Context, token, csrf string) (bool, error) {
	tokenHash := sha256.Sum256([]byte(token))
	csrfHash := sha256.Sum256([]byte(csrf))
	var valid bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE token_hash = $1 AND csrf_hash = $2 AND expires_at > now()
		)`, tokenHash[:], csrfHash[:]).Scan(&valid)
	return valid, err
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	tokenHash := sha256.Sum256([]byte(token))
	_, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE token_hash = $1", tokenHash[:])
	return err
}

func (s *Store) ListProducts(ctx context.Context, page, pageSize int, categoryID, search string) ([]Product, int, error) {
	args := []any{}
	where := []string{"p.active = true"}
	if categoryID != "" {
		args = append(args, categoryID)
		where = append(where, fmt.Sprintf("p.category_id = $%d", len(args)))
	}
	if search != "" {
		args = append(args, search)
		where = append(where, fmt.Sprintf("to_tsvector('english', p.name || ' ' || p.description) @@ plainto_tsquery('english', $%d)", len(args)))
	}
	filter := strings.Join(where, " AND ")
	var total int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM products p WHERE "+filter, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	query := fmt.Sprintf(`
		SELECT p.id::text, coalesce(p.category_id::text, ''), p.name, p.slug,
			p.description, p.image_url, p.price_cents, p.active,
			coalesce(i.quantity, 0), p.created_at
		FROM products p LEFT JOIN inventory i ON i.product_id = p.id
		WHERE %s ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d`, filter, len(args)-1, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	products := make([]Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, product)
	}
	return products, total, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProduct(row rowScanner) (Product, error) {
	var p Product
	var category string
	var cents int64
	err := row.Scan(&p.ID, &category, &p.Name, &p.Slug, &p.Description, &p.ImageURL,
		&cents, &p.Active, &p.Inventory, &p.CreatedAt)
	if category != "" {
		p.CategoryID = &category
	}
	p.Price = float64(cents) / 100
	return p, err
}

func (s *Store) GetProduct(ctx context.Context, id string) (Product, error) {
	p, err := scanProduct(s.pool.QueryRow(ctx, `
		SELECT p.id::text, coalesce(p.category_id::text, ''), p.name, p.slug,
			p.description, p.image_url, p.price_cents, p.active,
			coalesce(i.quantity, 0), p.created_at
		FROM products p LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.id = $1 AND p.active = true`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

func (s *Store) Categories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx, "SELECT id::text, name, slug, description, created_at FROM categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (s *Store) Cart(ctx context.Context, userID string) (Cart, error) {
	_, err := s.pool.Exec(ctx, "INSERT INTO carts (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING", userID)
	if err != nil {
		return Cart{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT ci.id::text, ci.product_id::text, p.name, p.image_url, p.price_cents, ci.quantity
		FROM carts c
		JOIN cart_items ci ON ci.cart_id = c.id
		JOIN products p ON p.id = ci.product_id
		WHERE c.user_id = $1 ORDER BY ci.created_at`, userID)
	if err != nil {
		return Cart{}, err
	}
	defer rows.Close()
	cart := Cart{Items: make([]CartItem, 0)}
	var totalCents int64
	for rows.Next() {
		var item CartItem
		var cents int64
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Name, &item.ImageURL, &cents, &item.Quantity); err != nil {
			return Cart{}, err
		}
		item.UnitPrice = float64(cents) / 100
		item.Subtotal = float64(cents*int64(item.Quantity)) / 100
		totalCents += cents * int64(item.Quantity)
		cart.Items = append(cart.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Cart{}, err
	}
	cart.Total = float64(totalCents) / 100
	return cart, nil
}

func (s *Store) AddCartItem(ctx context.Context, userID, productID string, quantity int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var active bool
	var available int
	err = tx.QueryRow(ctx, `
		SELECT p.active, coalesce(i.quantity, 0)
		FROM products p LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.id = $1`, productID).Scan(&active, &available)
	if errors.Is(err, pgx.ErrNoRows) || !active {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if quantity > available {
		return ErrInsufficientStock
	}
	var cartID string
	err = tx.QueryRow(ctx, `
		INSERT INTO carts (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = now()
		RETURNING id::text`, userID).Scan(&cartID)
	if err != nil {
		return err
	}
	var current int
	err = tx.QueryRow(ctx, "SELECT quantity FROM cart_items WHERE cart_id = $1 AND product_id = $2 FOR UPDATE", cartID, productID).Scan(&current)
	if err == nil && current+quantity > available {
		return ErrInsufficientStock
	}
	if err == nil && current+quantity > 99 {
		return ErrQuantityLimit
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, $3)
		ON CONFLICT (cart_id, product_id) DO UPDATE
		SET quantity = cart_items.quantity + EXCLUDED.quantity, updated_at = now()`,
		cartID, productID, quantity)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) UpdateCartItem(ctx context.Context, userID, itemID string, quantity int) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE cart_items ci SET quantity = $3, updated_at = now()
		FROM carts c, inventory i
		WHERE ci.cart_id = c.id AND c.user_id = $1 AND ci.id = $2
			AND ci.product_id = i.product_id AND i.quantity >= $3`, userID, itemID, quantity)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM cart_items ci JOIN carts c ON c.id = ci.cart_id
			WHERE c.user_id = $1 AND ci.id = $2)`, userID, itemID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrInsufficientStock
		}
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteCartItem(ctx context.Context, userID, itemID string) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM cart_items ci USING carts c
		WHERE ci.cart_id = c.id AND c.user_id = $1 AND ci.id = $2`, userID, itemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateOrder(ctx context.Context, userID string) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)
	var cartID string
	err = tx.QueryRow(ctx, "SELECT id::text FROM carts WHERE user_id = $1 FOR UPDATE", userID).Scan(&cartID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrEmptyCart
	}
	if err != nil {
		return Order{}, err
	}
	rows, err := tx.Query(ctx, `
		SELECT ci.product_id::text, p.name, p.price_cents, ci.quantity, i.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id AND p.active = true
		JOIN inventory i ON i.product_id = p.id
		WHERE ci.cart_id = $1
		ORDER BY ci.product_id
		FOR UPDATE OF ci, i, p`, cartID)
	if err != nil {
		return Order{}, err
	}
	type row struct {
		id       string
		name     string
		cents    int64
		quantity int
		stock    int
	}
	entries := make([]row, 0)
	var total int64
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.name, &item.cents, &item.quantity, &item.stock); err != nil {
			rows.Close()
			return Order{}, err
		}
		if item.quantity > item.stock {
			rows.Close()
			return Order{}, ErrInsufficientStock
		}
		if item.cents > 0 && int64(item.quantity) > math.MaxInt64/item.cents {
			rows.Close()
			return Order{}, errors.New("order total exceeds supported range")
		}
		total += item.cents * int64(item.quantity)
		entries = append(entries, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Order{}, err
	}
	rows.Close()
	if len(entries) == 0 {
		return Order{}, ErrEmptyCart
	}
	var order Order
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, total_cents) VALUES ($1, $2)
		RETURNING id::text, status, total_cents, created_at`, userID, total).
		Scan(&order.ID, &order.Status, &total, &order.CreatedAt)
	if err != nil {
		return Order{}, err
	}
	order.Total = float64(total) / 100
	order.UserID = userID
	order.Items = make([]OrderItem, 0, len(entries))
	for _, item := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, unit_price_cents, quantity)
			VALUES ($1, $2, $3, $4, $5)`, order.ID, item.id, item.name, item.cents, item.quantity); err != nil {
			return Order{}, err
		}
		if _, err := tx.Exec(ctx, "UPDATE inventory SET quantity = quantity - $2, updated_at = now() WHERE product_id = $1", item.id, item.quantity); err != nil {
			return Order{}, err
		}
		productID := item.id
		order.Items = append(order.Items, OrderItem{
			ProductID: &productID, Name: item.name,
			UnitPrice: float64(item.cents) / 100, Quantity: item.quantity,
		})
	}
	if _, err := tx.Exec(ctx, "DELETE FROM cart_items WHERE cart_id = $1", cartID); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return order, nil
}

func (s *Store) Orders(ctx context.Context, userID, orderID string, page, pageSize int) ([]Order, int, error) {
	args := []any{userID}
	where := "user_id = $1"
	if orderID != "" {
		args = append(args, orderID)
		where += " AND id = $2"
	}
	var total int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	query := fmt.Sprintf("SELECT id::text, user_id::text, status, total_cents, created_at FROM orders WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		where, len(args)-1, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	orders := make([]Order, 0)
	for rows.Next() {
		var order Order
		var cents int64
		if err := rows.Scan(&order.ID, &order.UserID, &order.Status, &cents, &order.CreatedAt); err != nil {
			return nil, 0, err
		}
		order.Total = float64(cents) / 100
		order.Items, err = s.orderItems(ctx, order.ID)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}
	return orders, total, rows.Err()
}

func (s *Store) orderItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT product_id::text, product_name, unit_price_cents, quantity
		FROM order_items WHERE order_id = $1 ORDER BY created_at`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		var id *string
		var cents int64
		if err := rows.Scan(&id, &item.Name, &cents, &item.Quantity); err != nil {
			return nil, err
		}
		item.ProductID = id
		item.UnitPrice = float64(cents) / 100
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) AdminCreateProduct(ctx context.Context, input ProductInput, cents int64) (Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Product{}, err
	}
	defer tx.Rollback(ctx)
	var id string
	var active bool
	if input.Active != nil {
		active = *input.Active
	} else {
		active = true
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO products (category_id, name, slug, description, image_url, price_cents, active)
		VALUES (nullif($1, '')::uuid, $2, $3, $4, $5, $6, $7)
		RETURNING id::text`, nullableID(input.CategoryID), input.Name, input.Slug, input.Description,
		input.ImageURL, cents, active).Scan(&id)
	if err != nil {
		return Product{}, err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO inventory (product_id, quantity) VALUES ($1, $2)", id, input.Inventory); err != nil {
		return Product{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Product{}, err
	}
	return s.adminProduct(ctx, id)
}

func (s *Store) AdminUpdateProduct(ctx context.Context, id string, input ProductInput, cents int64) (Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Product{}, err
	}
	defer tx.Rollback(ctx)
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	tag, err := tx.Exec(ctx, `
		UPDATE products SET category_id = nullif($2, '')::uuid, name = $3, slug = $4,
			description = $5, image_url = $6, price_cents = $7, active = $8, updated_at = now()
		WHERE id = $1`, id, nullableID(input.CategoryID), input.Name, input.Slug,
		input.Description, input.ImageURL, cents, active)
	if err != nil {
		return Product{}, err
	}
	if tag.RowsAffected() == 0 {
		return Product{}, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO inventory (product_id, quantity) VALUES ($1, $2)
		ON CONFLICT (product_id) DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = now()`,
		id, input.Inventory); err != nil {
		return Product{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Product{}, err
	}
	return s.adminProduct(ctx, id)
}

func nullableID(id *string) string {
	if id == nil {
		return ""
	}
	return *id
}

func (s *Store) adminProduct(ctx context.Context, id string) (Product, error) {
	product, err := scanProduct(s.pool.QueryRow(ctx, `
		SELECT p.id::text, coalesce(p.category_id::text, ''), p.name, p.slug,
			p.description, p.image_url, p.price_cents, p.active,
			coalesce(i.quantity, 0), p.created_at
		FROM products p LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return product, err
}

func (s *Store) AdminDeleteProduct(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "UPDATE products SET active = false, updated_at = now() WHERE id = $1 AND active = true", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AdminCreateCategory(ctx context.Context, input CategoryInput) (Category, error) {
	var category Category
	err := s.pool.QueryRow(ctx, `
		INSERT INTO categories (name, slug, description) VALUES ($1, $2, $3)
		RETURNING id::text, name, slug, description, created_at`, input.Name, input.Slug, input.Description).
		Scan(&category.ID, &category.Name, &category.Slug, &category.Description, &category.CreatedAt)
	return category, err
}

func (s *Store) AdminUpdateCategory(ctx context.Context, id string, input CategoryInput) (Category, error) {
	var category Category
	err := s.pool.QueryRow(ctx, `
		UPDATE categories SET name = $2, slug = $3, description = $4, updated_at = now()
		WHERE id = $1 RETURNING id::text, name, slug, description, created_at`,
		id, input.Name, input.Slug, input.Description).
		Scan(&category.ID, &category.Name, &category.Slug, &category.Description, &category.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	return category, err
}

func (s *Store) AdminDeleteCategory(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM categories WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AdminSetInventory(ctx context.Context, productID string, quantity int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO inventory (product_id, quantity) VALUES ($1, $2)
		ON CONFLICT (product_id) DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = now()`,
		productID, quantity)
	if err != nil {
		if strings.Contains(err.Error(), "inventory_product_id_fkey") {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Store) AdminOrders(ctx context.Context, page, pageSize int) ([]Order, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM orders").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, user_id::text, status, total_cents, created_at
		FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	orders := make([]Order, 0)
	for rows.Next() {
		var order Order
		var cents int64
		if err := rows.Scan(&order.ID, &order.UserID, &order.Status, &cents, &order.CreatedAt); err != nil {
			return nil, 0, err
		}
		order.Total = float64(cents) / 100
		order.Items, err = s.orderItems(ctx, order.ID)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}
	return orders, total, rows.Err()
}

func (s *Store) AdminUpdateOrderStatus(ctx context.Context, id, status string) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	var previousStatus string
	if err := tx.QueryRow(ctx, "SELECT status FROM orders WHERE id = $1 FOR UPDATE", id).Scan(&previousStatus); errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	} else if err != nil {
		return Order{}, err
	}
	if !validStatusTransition(previousStatus, status) {
		return Order{}, ErrInvalidTransition
	}
	if status == "cancelled" && previousStatus != "cancelled" {
		if _, err := tx.Exec(ctx, `
			UPDATE inventory i SET quantity = i.quantity + oi.quantity, updated_at = now()
			FROM order_items oi
			WHERE oi.order_id = $1 AND oi.product_id = i.product_id`, id); err != nil {
			return Order{}, err
		}
	}
	var order Order
	var cents int64
	err = tx.QueryRow(ctx, `
		UPDATE orders SET status = $2, updated_at = now() WHERE id = $1
		RETURNING id::text, user_id::text, status, total_cents, created_at`, id, status).
		Scan(&order.ID, &order.UserID, &order.Status, &cents, &order.CreatedAt)
	if err != nil {
		return Order{}, err
	}
	order.Total = float64(cents) / 100
	rows, err := tx.Query(ctx, `
		SELECT product_id::text, product_name, unit_price_cents, quantity
		FROM order_items WHERE order_id = $1 ORDER BY created_at`, order.ID)
	if err != nil {
		return Order{}, err
	}
	order.Items = make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		var productID *string
		var itemCents int64
		if err := rows.Scan(&productID, &item.Name, &itemCents, &item.Quantity); err != nil {
			rows.Close()
			return Order{}, err
		}
		item.ProductID = productID
		item.UnitPrice = float64(itemCents) / 100
		order.Items = append(order.Items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Order{}, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return order, nil
}

func validStatusTransition(current, next string) bool {
	switch current {
	case "pending":
		return next == "pending" || next == "processing" || next == "cancelled"
	case "processing":
		return next == "processing" || next == "shipped" || next == "cancelled"
	case "shipped":
		return next == "shipped" || next == "completed"
	case "completed":
		return next == "completed"
	case "cancelled":
		return next == "cancelled"
	default:
		return false
	}
}

type SalesReport struct {
	OrderCount int     `json:"order_count"`
	Revenue    float64 `json:"revenue"`
}

func (s *Store) AdminSalesReport(ctx context.Context) (SalesReport, error) {
	var report SalesReport
	var cents int64
	err := s.pool.QueryRow(ctx, `
		SELECT count(*), coalesce(sum(total_cents), 0)
		FROM orders WHERE status = 'completed'`).Scan(&report.OrderCount, &cents)
	report.Revenue = float64(cents) / 100
	return report, err
}
