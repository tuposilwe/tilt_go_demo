package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Restaurant struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Cuisine     string    `json:"cuisine"`
	Description string    `json:"description"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	PostalCode  string    `json:"postal_code"`
	Country     string    `json:"country"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Website     string    `json:"website"`
	PriceLevel  int       `json:"price_level"`
	Rating      float64   `json:"rating"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type restaurantInput struct {
	Name        *string  `json:"name"`
	Cuisine     *string  `json:"cuisine"`
	Description *string  `json:"description"`
	Address     *string  `json:"address"`
	City        *string  `json:"city"`
	State       *string  `json:"state"`
	PostalCode  *string  `json:"postal_code"`
	Country     *string  `json:"country"`
	Phone       *string  `json:"phone"`
	Email       *string  `json:"email"`
	Website     *string  `json:"website"`
	PriceLevel  *int     `json:"price_level"`
	Rating      *float64 `json:"rating"`
}

type restaurantAPI struct {
	db *sql.DB
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

func newRouter(db *sql.DB) http.Handler {
	api := restaurantAPI{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Restaurant API",
			"version": "1.0.0",
		})
	})
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /api/restaurants", api.list)
	mux.HandleFunc("POST /api/restaurants", api.create)
	mux.HandleFunc("GET /api/restaurants/{id}", api.get)
	mux.HandleFunc("PATCH /api/restaurants/{id}", api.update)
	mux.HandleFunc("DELETE /api/restaurants/{id}", api.delete)
	return mux
}

func (a restaurantAPI) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.db.PingContext(ctx); err != nil {
		log.Printf("database health check failed: %v", err)
		writeError(w, http.StatusServiceUnavailable, "database_unavailable", "database is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a restaurantAPI) list(w http.ResponseWriter, r *http.Request) {
	limit, err := parseQueryInt(r, "limit", 20, 1, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 100")
		return
	}
	offset, err := parseQueryInt(r, "offset", 0, 0, math.MaxInt32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_offset", "offset must be a non-negative integer")
		return
	}

	filters := make([]string, 0, 3)
	args := make([]any, 0, 5)
	for _, filter := range []struct {
		param  string
		column string
	}{
		{"city", "city"},
		{"cuisine", "cuisine"},
	} {
		if value := strings.TrimSpace(r.URL.Query().Get(filter.param)); value != "" {
			args = append(args, value)
			filters = append(filters, fmt.Sprintf("%s ILIKE $%d", filter.column, len(args)))
		}
	}
	if search := strings.TrimSpace(r.URL.Query().Get("q")); search != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)
		args = append(args, "%"+escaped+"%")
		filters = append(filters, fmt.Sprintf("(name ILIKE $%d ESCAPE E'\\\\' OR description ILIKE $%d ESCAPE E'\\\\')", len(args), len(args)))
	}

	where := ""
	if len(filters) > 0 {
		where = " WHERE " + strings.Join(filters, " AND ")
	}
	var total int
	if err := a.db.QueryRowContext(r.Context(), "SELECT count(*) FROM restaurants"+where, args...).Scan(&total); err != nil {
		log.Printf("count restaurants: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list restaurants")
		return
	}

	args = append(args, limit, offset)
	query := "SELECT " + restaurantColumns + " FROM restaurants" + where +
		fmt.Sprintf(" ORDER BY name, id LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := a.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("list restaurants: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list restaurants")
		return
	}
	defer rows.Close()

	restaurants := make([]Restaurant, 0)
	for rows.Next() {
		restaurant, err := scanRestaurant(rows)
		if err != nil {
			log.Printf("scan restaurant: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "could not list restaurants")
			return
		}
		restaurants = append(restaurants, restaurant)
	}
	if err := rows.Err(); err != nil {
		log.Printf("iterate restaurants: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list restaurants")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": restaurants,
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

func (a restaurantAPI) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	restaurant, err := scanRestaurant(a.db.QueryRowContext(r.Context(),
		"SELECT "+restaurantColumns+" FROM restaurants WHERE id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "restaurant not found")
		return
	}
	if err != nil {
		log.Printf("get restaurant %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not get restaurant")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Restaurant{"data": restaurant})
}

func (a restaurantAPI) create(w http.ResponseWriter, r *http.Request) {
	var input restaurantInput
	if !decodeInput(w, r, &input) {
		return
	}
	restaurant, err := input.toRestaurant()
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	created, err := scanRestaurant(a.db.QueryRowContext(r.Context(),
		`INSERT INTO restaurants
			(name, cuisine, description, address, city, state, postal_code, country, phone, email, website, price_level, rating)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING `+restaurantColumns,
		restaurant.Name, restaurant.Cuisine, restaurant.Description, restaurant.Address,
		restaurant.City, restaurant.State, restaurant.PostalCode, restaurant.Country,
		restaurant.Phone, restaurant.Email, restaurant.Website, restaurant.PriceLevel, restaurant.Rating))
	if err != nil {
		log.Printf("create restaurant: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create restaurant")
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/restaurants/%d", created.ID))
	writeJSON(w, http.StatusCreated, map[string]Restaurant{"data": created})
}

func (a restaurantAPI) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var input restaurantInput
	if !decodeInput(w, r, &input) {
		return
	}
	existing, err := scanRestaurant(a.db.QueryRowContext(r.Context(),
		"SELECT "+restaurantColumns+" FROM restaurants WHERE id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "restaurant not found")
		return
	}
	if err != nil {
		log.Printf("read restaurant %d for update: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update restaurant")
		return
	}

	updated, err := input.apply(existing)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	updated, err = scanRestaurant(a.db.QueryRowContext(r.Context(),
		`UPDATE restaurants
		 SET name = $1, cuisine = $2, description = $3, address = $4, city = $5,
		     state = $6, postal_code = $7, country = $8, phone = $9, email = $10,
		     website = $11, price_level = $12, rating = $13, updated_at = now()
		 WHERE id = $14
		 RETURNING `+restaurantColumns,
		updated.Name, updated.Cuisine, updated.Description, updated.Address,
		updated.City, updated.State, updated.PostalCode, updated.Country,
		updated.Phone, updated.Email, updated.Website, updated.PriceLevel, updated.Rating, id))
	if err != nil {
		log.Printf("update restaurant %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update restaurant")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Restaurant{"data": updated})
}

func (a restaurantAPI) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	result, err := a.db.ExecContext(r.Context(), "DELETE FROM restaurants WHERE id = $1", id)
	if err != nil {
		log.Printf("delete restaurant %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete restaurant")
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		log.Printf("check deleted restaurant %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete restaurant")
		return
	}
	if count == 0 {
		writeError(w, http.StatusNotFound, "not_found", "restaurant not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (input restaurantInput) toRestaurant() (Restaurant, error) {
	var restaurant Restaurant
	required := []struct {
		name string
		dest *string
		src  *string
	}{
		{"name", &restaurant.Name, input.Name},
		{"cuisine", &restaurant.Cuisine, input.Cuisine},
		{"address", &restaurant.Address, input.Address},
		{"city", &restaurant.City, input.City},
		{"country", &restaurant.Country, input.Country},
	}
	for _, field := range required {
		if field.src == nil {
			return Restaurant{}, fmt.Errorf("%s is required", field.name)
		}
		*field.dest = strings.TrimSpace(*field.src)
	}
	restaurant.Description = valueOrEmpty(input.Description)
	restaurant.State = valueOrEmpty(input.State)
	restaurant.PostalCode = valueOrEmpty(input.PostalCode)
	restaurant.Phone = valueOrEmpty(input.Phone)
	restaurant.Email = valueOrEmpty(input.Email)
	restaurant.Website = valueOrEmpty(input.Website)
	if input.PriceLevel != nil {
		restaurant.PriceLevel = *input.PriceLevel
	} else {
		restaurant.PriceLevel = 2
	}
	if input.Rating != nil {
		restaurant.Rating = *input.Rating
	}
	return restaurant, validateRestaurant(restaurant)
}

func (input restaurantInput) apply(existing Restaurant) (Restaurant, error) {
	setString := func(target *string, value *string) {
		if value != nil {
			*target = strings.TrimSpace(*value)
		}
	}
	setString(&existing.Name, input.Name)
	setString(&existing.Cuisine, input.Cuisine)
	setString(&existing.Description, input.Description)
	setString(&existing.Address, input.Address)
	setString(&existing.City, input.City)
	setString(&existing.State, input.State)
	setString(&existing.PostalCode, input.PostalCode)
	setString(&existing.Country, input.Country)
	setString(&existing.Phone, input.Phone)
	setString(&existing.Email, input.Email)
	setString(&existing.Website, input.Website)
	if input.PriceLevel != nil {
		existing.PriceLevel = *input.PriceLevel
	}
	if input.Rating != nil {
		existing.Rating = *input.Rating
	}
	return existing, validateRestaurant(existing)
}

func validateRestaurant(restaurant Restaurant) error {
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{"name", restaurant.Name, 120},
		{"cuisine", restaurant.Cuisine, 80},
		{"description", restaurant.Description, 1000},
		{"address", restaurant.Address, 200},
		{"city", restaurant.City, 100},
		{"state", restaurant.State, 100},
		{"postal_code", restaurant.PostalCode, 20},
		{"country", restaurant.Country, 100},
		{"phone", restaurant.Phone, 40},
		{"email", restaurant.Email, 254},
		{"website", restaurant.Website, 2048},
	} {
		if field.name == "name" || field.name == "cuisine" || field.name == "address" || field.name == "city" || field.name == "country" {
			if strings.TrimSpace(field.value) == "" {
				return fmt.Errorf("%s is required", field.name)
			}
		}
		if utf8.RuneCountInString(field.value) > field.max {
			return fmt.Errorf("%s must be at most %d characters", field.name, field.max)
		}
	}
	if restaurant.PriceLevel < 1 || restaurant.PriceLevel > 4 {
		return fmt.Errorf("price_level must be between 1 and 4")
	}
	if math.IsNaN(restaurant.Rating) || math.IsInf(restaurant.Rating, 0) || restaurant.Rating < 0 || restaurant.Rating > 5 {
		return fmt.Errorf("rating must be between 0 and 5")
	}
	if restaurant.Email != "" {
		address, err := mail.ParseAddress(restaurant.Email)
		if err != nil || address.Address != restaurant.Email {
			return fmt.Errorf("email must be a valid email address")
		}
	}
	if restaurant.Website != "" {
		parsed, err := url.ParseRequestURI(restaurant.Website)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("website must be a valid HTTP or HTTPS URL")
		}
	}
	return nil
}

func decodeInput(w http.ResponseWriter, r *http.Request, input *restaurantInput) bool {
	contentType, _, contentTypeErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if contentTypeErr != nil || contentType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be a valid restaurant JSON object")
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain exactly one JSON object")
		return false
	}
	return true
}

func parseQueryInt(r *http.Request, key string, fallback, min, max int) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || parsed > max {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return parsed, nil
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_id", "restaurant id must be a positive integer")
		return 0, false
	}
	return id, true
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message}})
}

const restaurantColumns = `id, name, cuisine, description, address, city, state, postal_code,
	country, phone, email, website, price_level, rating, created_at, updated_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRestaurant(row rowScanner) (Restaurant, error) {
	var restaurant Restaurant
	err := row.Scan(
		&restaurant.ID, &restaurant.Name, &restaurant.Cuisine, &restaurant.Description,
		&restaurant.Address, &restaurant.City, &restaurant.State, &restaurant.PostalCode,
		&restaurant.Country, &restaurant.Phone, &restaurant.Email, &restaurant.Website,
		&restaurant.PriceLevel, &restaurant.Rating, &restaurant.CreatedAt, &restaurant.UpdatedAt,
	)
	return restaurant, err
}
