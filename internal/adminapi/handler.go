package adminapi

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultLimit = 100
	maxLimit     = 500
)

// devTenantID is the deterministic tenant UUID from seeds/00_tenants_dev.sql.
// Acts as the implicit X-Tenant-Id when the header is absent in local dev.
const devTenantID = "00000000-0000-5000-8000-000000000001"

// Handler holds the shared pgxpool. One instance per process.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler constructs a Handler bound to the given pool.
func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

// ─── routes ────────────────────────────────────────────────────────────────

// Register attaches all admin routes to the given gin engine under /v1/admin.
// Phase 2 = read-only; Phase 3 adds POST/PUT/DELETE.
func (h *Handler) Register(r *gin.Engine) {
	g := r.Group("/v1/admin")
	g.GET("/_schemas", h.listSchemas)
	g.GET("/:table", h.list)
	g.GET("/:table/:id", h.get)
	g.POST("/:table", h.post)
	g.PUT("/:table/:id", h.put)
	g.DELETE("/:table/:id", h.del)
}

// ─── handlers ──────────────────────────────────────────────────────────────

// listSchemas returns the full catalogue — diagnostics + smoke source.
func (h *Handler) listSchemas(c *gin.Context) {
	schemas := AllSchemas()
	sort.Slice(schemas, func(i, j int) bool { return schemas[i].URL < schemas[j].URL })
	out := make([]gin.H, 0, len(schemas))
	for _, s := range schemas {
		out = append(out, gin.H{
			"url":     s.URL,
			"table":   s.Table,
			"pk":      s.PK,
			"tenant":  s.TenantColumn != "",
			"mutable": s.Mutable,
			"filters": filterKeys(s.AllowedFilters),
		})
	}
	c.JSON(http.StatusOK, gin.H{"count": len(out), "schemas": out})
}

// list — GET /v1/admin/{table}?limit=N&offset=M&{filter}={value}
func (h *Handler) list(c *gin.Context) {
	schema, ok := SchemaByURL(c.Param("table"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown table"})
		return
	}

	limit, offset := parseLimitOffset(c)
	tenantID := tenantFromHeader(c)

	sql, args := buildSelectSQL(schema, tenantID, c.Request.URL.Query(), limit, offset)

	rows, err := h.pool.Query(c.Request.Context(), sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "sql": sql})
		return
	}
	defer rows.Close()

	items, err := collectRowsAsMap(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(items), "items": items})
}

// get — GET /v1/admin/{table}/{id}
func (h *Handler) get(c *gin.Context) {
	schema, ok := SchemaByURL(c.Param("table"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown table"})
		return
	}
	id := c.Param("id")
	tenantID := tenantFromHeader(c)

	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", schema.Table, schema.PK)
	args := []any{id}
	if schema.TenantColumn != "" {
		sql += fmt.Sprintf(" AND %s = $2", schema.TenantColumn)
		args = append(args, tenantID)
	}
	sql += " LIMIT 1"

	rows, err := h.pool.Query(c.Request.Context(), sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	items, err := collectRowsAsMap(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, items[0])
}

// post — POST /v1/admin/{table} (Phase 3 wires writes; here returns 501).
func (h *Handler) post(c *gin.Context) {
	schema, ok := SchemaByURL(c.Param("table"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown table"})
		return
	}
	if !schema.Mutable {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": fmt.Sprintf("%s is read-only", schema.Table)})
		return
	}
	if err := h.insertViaJSON(c.Request.Context(), schema, c); err != nil {
		c.JSON(httpStatusForErr(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "created", "table": schema.Table})
}

// put — PUT /v1/admin/{table}/{id}
func (h *Handler) put(c *gin.Context) {
	schema, ok := SchemaByURL(c.Param("table"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown table"})
		return
	}
	if !schema.Mutable {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": fmt.Sprintf("%s is read-only", schema.Table)})
		return
	}
	id := c.Param("id")
	if err := h.updateViaJSON(c.Request.Context(), schema, id, c); err != nil {
		c.JSON(httpStatusForErr(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated", "table": schema.Table, "id": id})
}

// del — DELETE /v1/admin/{table}/{id}
func (h *Handler) del(c *gin.Context) {
	schema, ok := SchemaByURL(c.Param("table"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown table"})
		return
	}
	if !schema.Mutable {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": fmt.Sprintf("%s is read-only", schema.Table)})
		return
	}
	id := c.Param("id")
	tenantID := tenantFromHeader(c)

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", schema.Table, schema.PK)
	args := []any{id}
	if schema.TenantColumn != "" {
		sql += fmt.Sprintf(" AND %s = $2", schema.TenantColumn)
		args = append(args, tenantID)
	}
	tag, err := h.pool.Exec(c.Request.Context(), sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted", "table": schema.Table, "id": id})
}

// ─── helpers ───────────────────────────────────────────────────────────────

func buildSelectSQL(s Schema, tenantID string, q map[string][]string, limit, offset int) (string, []any) {
	var (
		parts = []string{fmt.Sprintf("SELECT * FROM %s", s.Table)}
		where = make([]string, 0, 4)
		args  = make([]any, 0, 4)
	)
	if s.TenantColumn != "" {
		where = append(where, fmt.Sprintf("%s = $%d", s.TenantColumn, len(args)+1))
		args = append(args, tenantID)
	}
	// Deterministic iteration for stable SQL across runs (helps caching + tests).
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		col, allowed := s.AllowedFilters[k]
		if !allowed {
			continue
		}
		vs := q[k]
		if len(vs) == 0 || vs[0] == "" {
			continue
		}
		where = append(where, fmt.Sprintf("%s = $%d", col, len(args)+1))
		args = append(args, vs[0])
	}
	if len(where) > 0 {
		parts = append(parts, "WHERE "+strings.Join(where, " AND "))
	}
	if s.DefaultOrder != "" {
		parts = append(parts, "ORDER BY "+s.DefaultOrder)
	}
	parts = append(parts, fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset))
	return strings.Join(parts, " "), args
}

func parseLimitOffset(c *gin.Context) (int, int) {
	limit := defaultLimit
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= maxLimit {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func tenantFromHeader(c *gin.Context) string {
	if v := c.GetHeader("X-Tenant-Id"); v != "" {
		return v
	}
	return devTenantID
}

func filterKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// collectRowsAsMap reads each row into map[string]any. Each value is
// normalised so JSON marshalling produces strings for UUIDs + numerics
// (instead of [16]byte arrays + exponentials).
func collectRowsAsMap(rows pgx.Rows) ([]map[string]any, error) {
	out := make([]map[string]any, 0, 16)
	fields := rows.FieldDescriptions()
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		m := make(map[string]any, len(fields))
		for i, f := range fields {
			m[string(f.Name)] = normaliseValue(vals[i])
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// normaliseValue converts pgx-native types to JSON-friendly forms.
//   - [16]byte → uuid hex string ("xxxxxxxx-xxxx-...")
//   - driver.Valuer (pgtype.Numeric etc) → string preserving precision
//   - everything else passes through (gin handles JSONB, time.Time, []string, etc)
func normaliseValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case [16]byte:
		u, err := uuid.FromBytes(x[:])
		if err != nil {
			return x // fall back to raw bytes (gin will b64)
		}
		return u.String()
	case driver.Valuer:
		dv, err := x.Value()
		if err != nil {
			return fmt.Sprintf("<invalid: %v>", err)
		}
		return dv
	}
	return v
}

// httpStatusForErr maps internal errors to HTTP status codes.
func httpStatusForErr(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, errBadRequest) {
		return http.StatusBadRequest
	}
	if errors.Is(err, errConflict) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

// ─── stubs for write methods (Phase 3) ─────────────────────────────────────

// insertViaJSON parses request body and INSERTs via raw SQL using only the
// schema's known columns (introspected once at handler startup).
func (h *Handler) insertViaJSON(ctx context.Context, s Schema, c *gin.Context) error {
	var body map[string]any
	if err := c.BindJSON(&body); err != nil {
		return fmt.Errorf("%w: invalid JSON: %v", errBadRequest, err)
	}
	cols, err := h.columnsFor(ctx, s.Table)
	if err != nil {
		return err
	}
	// Tenant scoping: if schema is tenant-scoped and caller didn't supply,
	// default to the dev tenant for convenience.
	if s.TenantColumn != "" {
		if _, present := body[s.TenantColumn]; !present {
			body[s.TenantColumn] = tenantFromHeader(c)
		}
	}
	insertCols := make([]string, 0, len(body))
	placeholders := make([]string, 0, len(body))
	args := make([]any, 0, len(body))
	for col := range body {
		if !cols[col] {
			continue // silently drop unknown columns to avoid SQL injection via key
		}
		insertCols = append(insertCols, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
		args = append(args, body[col])
	}
	if len(insertCols) == 0 {
		return fmt.Errorf("%w: no recognised columns in body", errBadRequest)
	}
	// Intentionally NO sort: placeholders $N reference args POSITION, so sorting
	// would misalign $N with args[N-1]. Map iteration is non-deterministic but
	// each (col, placeholder, arg) triple is built together so they stay aligned.
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", s.Table, strings.Join(insertCols, ", "), strings.Join(placeholders, ", "))
	if _, err := h.pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", errConflict, err)
	}
	return nil
}

// updateViaJSON parses body and UPDATEs the row identified by {id} (+ tenant).
func (h *Handler) updateViaJSON(ctx context.Context, s Schema, id string, c *gin.Context) error {
	var body map[string]any
	if err := c.BindJSON(&body); err != nil {
		return fmt.Errorf("%w: invalid JSON: %v", errBadRequest, err)
	}
	cols, err := h.columnsFor(ctx, s.Table)
	if err != nil {
		return err
	}
	sets := make([]string, 0, len(body))
	args := make([]any, 0, len(body)+2)
	for col, val := range body {
		if !cols[col] || col == s.PK {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)+1))
		args = append(args, val)
	}
	if len(sets) == 0 {
		return fmt.Errorf("%w: no updatable fields in body", errBadRequest)
	}
	// No sort — $N placeholders are positional, sorting would misalign args.
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d", s.Table, strings.Join(sets, ", "), s.PK, len(args)+1)
	args = append(args, id)
	if s.TenantColumn != "" {
		sql += fmt.Sprintf(" AND %s = $%d", s.TenantColumn, len(args)+1)
		args = append(args, tenantFromHeader(c))
	}
	tag, err := h.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%w: %v", errConflict, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: not found", errBadRequest)
	}
	return nil
}

// columnsFor introspects the schema once + caches the column set per table.
func (h *Handler) columnsFor(ctx context.Context, table string) (map[string]bool, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
	`, table)
	if err != nil {
		return nil, fmt.Errorf("introspect %s: %w", table, err)
	}
	defer rows.Close()
	out := make(map[string]bool, 32)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, nil
}

// ─── sentinel errors ───────────────────────────────────────────────────────

var (
	errBadRequest = errors.New("bad request")
	errConflict   = errors.New("conflict")
)
