package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db     dBInterface
	rootDB *sqlx.DB
}

type dBInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		rootDB: db,
		db:     db,
	}
}

func (r *Repository) GetDevice(ctx context.Context, id DeviceID) (*Device, error) {
	device := new(Device)

	query := `
		SELECT 
		    d.id,
			d.name,
			d.created_at,
			d.deleted_at,
			k.key_prefix
		FROM devices d
        INNER JOIN device_api_keys k ON d.id = k.device_id
		WHERE d.id = ? AND d.deleted_at IS NULL`

	err := r.db.GetContext(ctx, device, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return device, nil
}

func (r *Repository) CreateDevice(ctx context.Context, params CreateDeviceParams) (*Device, error) {
	now := time.Now().UTC()
	var createdDevice *Device

	err := r.runInTx(ctx, func(tx *Repository) error {
		// Create device
		deviceQuery := `
		INSERT INTO devices (name, created_at)
		VALUES (?, ?) RETURNING id
		`
		var createdDeviceID DeviceID
		err := tx.db.GetContext(ctx, &createdDeviceID, deviceQuery, params.Name, now)
		if err != nil {
			if domainErr, ok := mapDeviceNameUniqueConstraintError(err); ok {
				return domainErr
			}
			return fmt.Errorf("insert device: %w", err)
		}

		// Add API KEY to device
		apiQuery := `
		INSERT INTO device_api_keys (device_id, key_prefix, key_hash, created_at)
		VALUES (?, ?, ?, ?)
	`

		_, err = tx.db.ExecContext(ctx, apiQuery, createdDeviceID, params.KeyPrefix, params.KeyHash, now)
		if err != nil {
			return err
		}

		createdDevice, err = tx.GetDevice(ctx, createdDeviceID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return createdDevice, nil
}

func (r *Repository) UpdateAPIKey(ctx context.Context, deviceID DeviceID, keyHash string, keyPrefix string) error {
	query := `UPDATE device_api_keys SET key_hash = ?, key_prefix = ?, created_at = CURRENT_TIMESTAMP WHERE device_id = ?`
	result, err := r.db.ExecContext(ctx, query, keyHash, keyPrefix, deviceID)
	if err != nil {
		return fmt.Errorf("update api key: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update api key rows affected: %w", err)
	}
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (r *Repository) CreateAddress(ctx context.Context, params CreateAddressParams) (*Address, error) {
	var address *Address

	err := r.runInTx(ctx, func(tx *Repository) error {
		query := `
		INSERT INTO addresses (device_id, ip, created_at)
		VALUES (?, ?, ?) RETURNING id
	`
		var addressID AddressID
		err := tx.db.GetContext(ctx, &addressID, query, params.DeviceID, params.IP.String(), time.Now().UTC())
		if err != nil {
			return err
		}

		//TODO: We shoudn't update the address itself, only record the event. This probably means we need to extract
		// the part that updates the address from the one that creates the event
		address, err = tx.recordAddressEvent(ctx, addressID, true, EventSourceManual)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("create address: %w", err)
	}
	return address, nil
}

func (r *Repository) GetAddressForDeviceByIP(ctx context.Context, deviceID DeviceID, ip netip.Addr) (*Address, error) {
	address := new(Address)

	query := `
		SELECT a.id,
		       a.device_id,
		       a.ip,
		       a.is_enabled,
		       a.source,
		       a.created_at,
		       a.updated_at
		FROM addresses a
		WHERE a.device_id = ?
		AND a.ip = ?
		ORDER BY a.updated_at DESC
	`

	err := r.db.GetContext(ctx, address, query, deviceID, ip.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address: %w", err)
	}

	return address, nil
}

func (r *Repository) CheckAddressOwnership(ctx context.Context, deviceID DeviceID, addressID AddressID) error {
	var dummy int

	query := `SELECT 1 FROM addresses WHERE id = ? AND device_id = ?`

	err := r.db.GetContext(ctx, &dummy, query, addressID, deviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAddressNotOwnedByDevice
		}
		return fmt.Errorf("failed to check address ownership: %w", err)
	}
	return nil
}

func (r *Repository) GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error) {
	device := new(Device)

	query := `
		SELECT d.* FROM devices d
		INNER JOIN device_api_keys k ON d.id = k.device_id
		WHERE k.key_hash = ? AND d.deleted_at IS NULL
	`

	err := r.db.GetContext(ctx, device, query, keyHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to get device by api key hash: %w", err)
	}

	return device, nil
}

func mapDeviceNameUniqueConstraintError(err error) (error, bool) {
	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "unique constraint failed") {
		return nil, false
	}
	if strings.Contains(message, "name") || strings.Contains(message, "idx_devices_name_active") {
		return ErrDuplicateDeviceName, true
	}
	return nil, false
}

func (r *Repository) DeleteDevice(ctx context.Context, deviceID DeviceID) error {
	query := `UPDATE devices SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now().UTC(), deviceID)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete device check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (r *Repository) GetEnabledIPEntries(ctx context.Context) ([]IPEntry, error) {
	var entries []IPEntry

	query := `
		SELECT a.ip, a.device_id, a.id AS address_id
		FROM addresses a
		WHERE a.is_enabled = 1
		GROUP BY a.ip
		HAVING a.updated_at = MAX(a.updated_at)
	`

	err := r.db.SelectContext(ctx, &entries, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled IP entries: %w", err)
	}

	if entries == nil {
		return []IPEntry{}, nil
	}

	return entries, nil
}

// GetAddress returns the current state for a single address ID.
func (r *Repository) GetAddress(ctx context.Context, addressID AddressID) (*Address, error) {
	state := new(Address)

	query := `
		SELECT a.id,
		       a.device_id,
		       a.ip,
		       a.is_enabled,
		       a.source,
		       a.created_at,
		       a.updated_at
		FROM addresses a
		WHERE a.id = ?
	`

	err := r.db.GetContext(ctx, state, query, addressID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get address current state: %w", err)
	}

	return state, nil
}

func (r *Repository) DisableAddress(ctx context.Context, addressID AddressID) (*Address, error) {
	return r.recordAddressEvent(ctx, addressID, false, EventSourceManual)
}

func (r *Repository) DisableAddresses(ctx context.Context, addressIDs []AddressID, source EventSource) ([]Address, error) {
	if len(addressIDs) == 0 {
		return []Address{}, nil
	}

	disabledAddresses := make([]Address, len(addressIDs))

	err := r.runInTx(ctx, func(tx *Repository) error {
		for i, addressID := range addressIDs {
			disabledAddress, err := tx.recordAddressEvent(ctx, addressID, false, source)
			if err != nil {
				return fmt.Errorf("failed to disable address %d: %w", addressID, err)
			}
			disabledAddresses[i] = *disabledAddress
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return disabledAddresses, nil
}

func (r *Repository) EnableAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error) {
	return r.recordAddressEvent(ctx, addressID, true, source)
}

// RefreshAddress records activity for an already-enabled address (same DB work as EnableAddress; used for semantic distinction).
// Refresh is modeled separately at the domain level, but persisted the same as enable to keep full audit history.
func (r *Repository) RefreshAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error) {
	return r.EnableAddress(ctx, addressID, source)
}

func (r *Repository) recordAddressEvent(ctx context.Context, addressID AddressID, isEnabled bool, source EventSource) (*Address, error) {
	var finalAddress *Address
	err := r.runInTx(ctx, func(tx *Repository) error {
		now := time.Now().UTC()

		insertEvent := `
		INSERT INTO address_events (address_id, is_enabled, source, created_at)
		VALUES (?, ?, ?, ?)
	`

		if _, err := tx.db.ExecContext(ctx, insertEvent, addressID, isEnabled, source, now); err != nil {
			return fmt.Errorf("failed to record event: %w", err)
		}

		updateState := `
		UPDATE addresses SET is_enabled = ?, source = ?, updated_at = ? WHERE id = ?
	`

		if _, err := tx.db.ExecContext(ctx, updateState, isEnabled, source, now, addressID); err != nil {
			return fmt.Errorf("failed to update address state: %w", err)
		}

		var err error
		finalAddress, err = tx.GetAddress(ctx, addressID)
		if err != nil {
			return fmt.Errorf("failed to get address current state: %w", err)

		}

		return nil

	})
	if err != nil {
		return nil, err
	}

	return finalAddress, nil
}

// strftimeFmt returns the SQLite strftime format for the given granularity.
func strftimeFmt(g Granularity) string {
	if g == GranularityDay {
		return "%Y-%m-%dT00:00:00Z"
	}
	return "%Y-%m-%dT%H:00:00Z"
}

// deviceIDPlaceholders builds an IN clause fragment and args for a slice of device IDs.
func deviceIDPlaceholders(ids []DeviceID) (string, []any) {
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	return strings.Join(placeholders, ", "), args
}

// escapeLIKE escapes SQL LIKE wildcards (% and _) in user input.
func escapeLIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// buildHistoryWhere builds the shared WHERE clause for both buckets and events queries.
// Returns the filter strings and args. The caller is responsible for joining them.
func buildHistoryWhere(q AddressHistoryQuery) ([]string, []any) {
	filters := []string{"d.deleted_at IS NULL"}
	var args []any

	if len(q.DeviceIDs) > 0 {
		in, idArgs := deviceIDPlaceholders(q.DeviceIDs)
		filters = append(filters, "a.device_id IN ("+in+")")
		args = append(args, idArgs...)
	}

	if !q.From.IsZero() {
		filters = append(filters, "aev.created_at >= ?")
		args = append(args, q.From)
	}
	if !q.To.IsZero() {
		filters = append(filters, "aev.created_at <= ?")
		args = append(args, q.To)
	}

	if q.Source != nil {
		filters = append(filters, "aev.source = ?")
		args = append(args, *q.Source)
	}
	if q.IsEnabled != nil {
		filters = append(filters, "aev.is_enabled = ?")
		args = append(args, *q.IsEnabled)
	}
	if q.IP != nil {
		filters = append(filters, "a.ip LIKE ? ESCAPE '\\'")
		args = append(args, "%"+escapeLIKE(*q.IP)+"%")
	}

	return filters, args
}

func joinWhere(filters []string) string {
	if len(filters) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(filters, " AND ")
}

func (r *Repository) GetAddressHistory(ctx context.Context, q AddressHistoryQuery) (AddressHistory, error) {
	filters, baseArgs := buildHistoryWhere(q)

	// ── Buckets ──────────────────────────────────────────────────────────
	bucketsQuery := `
		SELECT
			strftime(?, aev.created_at) AS bucket,
			COUNT(DISTINCT CASE WHEN aev.is_enabled THEN aev.address_id END) AS active_count,
			COUNT(*) AS event_count
		FROM address_events aev
		JOIN addresses a ON a.id = aev.address_id
		JOIN devices d ON d.id = a.device_id
	` + joinWhere(filters) + `
		GROUP BY bucket
		ORDER BY bucket ASC
	`

	bucketArgs := make([]any, 0, 1+len(baseArgs))
	bucketArgs = append(bucketArgs, strftimeFmt(q.Granularity))
	bucketArgs = append(bucketArgs, baseArgs...)

	var buckets []AddressEventBucket
	if err := r.db.SelectContext(ctx, &buckets, bucketsQuery, bucketArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("get history buckets: %w", err)
	}
	if buckets == nil {
		buckets = []AddressEventBucket{}
	}

	// ── Events (paginated) ───────────────────────────────────────────────
	// Count total (without cursor)
	countQuery := `
		SELECT COUNT(*)
		FROM address_events aev
		JOIN addresses a ON a.id = aev.address_id
		JOIN devices d ON d.id = a.device_id
	` + joinWhere(filters)

	var totalEvents int
	if err := r.db.GetContext(ctx, &totalEvents, countQuery, baseArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("count history events: %w", err)
	}

	// Select with cursor + limit
	eventFilters := make([]string, len(filters))
	copy(eventFilters, filters)
	eventArgs := make([]any, len(baseArgs))
	copy(eventArgs, baseArgs)

	if q.BeforeID != nil {
		eventFilters = append(eventFilters, "aev.id < ?")
		eventArgs = append(eventArgs, *q.BeforeID)
	}
	eventArgs = append(eventArgs, q.Limit)

	eventsQuery := `
		SELECT aev.id, aev.created_at, a.ip, aev.is_enabled, aev.source,
		       a.device_id, d.name AS device_name
		FROM address_events aev
		JOIN addresses a ON a.id = aev.address_id
		JOIN devices d ON d.id = a.device_id
	` + joinWhere(eventFilters) + `
		ORDER BY aev.id DESC
		LIMIT ?
	`

	var events []AddressStateChange
	if err := r.db.SelectContext(ctx, &events, eventsQuery, eventArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("get history events: %w", err)
	}
	if events == nil {
		events = []AddressStateChange{}
	}

	return AddressHistory{
		Buckets:     buckets,
		Events:      events,
		TotalEvents: totalEvents,
	}, nil
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) runInTx(ctx context.Context, fn func(*Repository) error) error {
	logger := slog.Default()
	if r.rootDB == nil {
		// We are already in a transaction. Do not nest it.
		return fn(r)
	}

	// Start the transaction
	tx, err := r.rootDB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	// Defer rollback (standard practice)
	defer func() {
		//nolint:staticcheck // Empty branch is intentional - ErrTxDone is expected after commit
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			// Rollback error is only significant if transaction wasn't already committed/rolled back
			logger.Error("failed to rollback transaction", slog.Any(logging.AttrKeyError, err))
		}
	}()

	// Create a COPY of the repository
	// We replace 'db' with the transaction 'tx' and set the rootDB to nil so that it is not reused
	txRepo := &Repository{
		rootDB: nil, // Prevent nested transactions
		db:     tx,  // All queries using txRepo.dbtmp will now use this transaction
	}

	// Run the business logic with the transactional repo
	if err := fn(txRepo); err != nil {
		return err
	}

	// Commit if successful
	return tx.Commit()
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) RunInTx(ctx context.Context, fn func(repository) error) error {
	return r.runInTx(ctx, func(repo *Repository) error {
		return fn(repo)
	})
}
