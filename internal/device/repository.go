package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) GetDevice(ctx context.Context, id DeviceID) (*Device, error) {
	device := new(Device)

	query := `
			SELECT
				d.id,
				d.name,
				d.device_type,
				d.description,
				d.icon,
				d.created_at,
				d.updated_at,
				d.deleted_at,
				k.key_prefix,
				(SELECT MAX(a.updated_at) FROM addresses a WHERE a.device_id = d.id) AS last_seen_at,
				d.owner_id AS owner_id
			FROM devices d
			LEFT JOIN device_api_keys k ON d.id = k.device_id
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
	createdDevice := new(Device)

	deviceType := params.DeviceType
	if deviceType == "" {
		deviceType = "static"
	}

	deviceQuery := `INSERT INTO devices (name, owner_id, device_type, created_at) VALUES (?, ?, ?, ?) RETURNING *`

	err := r.db.GetContext(ctx, createdDevice, deviceQuery, params.Name, params.OwnerID, deviceType, now)
	if err != nil {
		if domainErr, ok := mapDeviceNameUniqueConstraintError(err); ok {
			return nil, domainErr
		}
		if domainErr, ok := mapOwnerFKConstraintError(err); ok {
			return nil, domainErr
		}
		return nil, fmt.Errorf("insert device: %w", err)
	}

	return createdDevice, nil
}

func (r *Repository) UpsertAPIKey(ctx context.Context, deviceID DeviceID, keyHash string, keyPrefix string) error {
	query := `
		INSERT INTO device_api_keys (device_id, key_prefix, key_hash, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(device_id) DO UPDATE SET
			key_prefix = excluded.key_prefix,
			key_hash   = excluded.key_hash,
			created_at = CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query, deviceID, keyPrefix, keyHash)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed") {
			return ErrDeviceNotFound
		}
		return fmt.Errorf("upsert api key: %w", err)
	}
	return nil
}

func (r *Repository) DeleteAPIKey(ctx context.Context, deviceID DeviceID) error {
	query := `DELETE FROM device_api_keys WHERE device_id = ?`
	result, err := r.db.ExecContext(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete api key rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNoAPIKey
	}
	return nil
}

func (r *Repository) CreateAddress(ctx context.Context, params CreateAddressParams, source EventSource) (*Address, error) {
	var address *Address

	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		now := time.Now().UTC()

		query := `
		INSERT INTO addresses (device_id, ip, is_enabled, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id
	`
		var addressID AddressID
		err := r.db.GetContext(ctx, &addressID, query, params.DeviceID, params.IP.String(), true, source, now, now)
		if err != nil {
			return err
		}

		if err := r.insertAddressEvent(ctx, addressID, true, source, now); err != nil {
			return err
		}

		address, err = r.GetAddress(ctx, addressID)
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
			SELECT d.id, d.name, d.device_type, d.description, d.icon, d.created_at, d.updated_at, d.deleted_at,
				   k.key_prefix,
				   d.owner_id AS owner_id
			FROM devices d
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

func mapOwnerFKConstraintError(err error) (error, bool) {
	if strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed") {
		return ErrOwnerNotFound, true
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

func (r *Repository) UpdateDevice(ctx context.Context, device *Device) (*Device, error) {
	query := `
		UPDATE devices
		SET name = ?, device_type = ?, description = ?, icon = ?, owner_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query,
		device.Name, string(device.DeviceType), device.Description, device.Icon, device.OwnerID, device.ID,
	)
	if err != nil {
		if domainErr, ok := mapDeviceNameUniqueConstraintError(err); ok {
			return nil, domainErr
		}
		if domainErr, ok := mapOwnerFKConstraintError(err); ok {
			return nil, domainErr
		}
		return nil, fmt.Errorf("update device: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update device rows affected: %w", err)
	}
	if rows == 0 {
		return nil, ErrDeviceNotFound
	}
	return r.GetDevice(ctx, device.ID)
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

// GetEnabledAddressesForDevice returns all enabled addresses for a device, ordered by updated_at DESC.
func (r *Repository) GetEnabledAddressesForDevice(ctx context.Context, deviceID DeviceID) ([]Address, error) {
	var addresses []Address

	const query = `
		SELECT id, device_id, ip, is_enabled, source, created_at, updated_at
		FROM addresses
		WHERE device_id = ? AND is_enabled = 1
		ORDER BY updated_at DESC
	`

	if err := r.db.SelectContext(ctx, &addresses, query, deviceID); err != nil {
		return nil, fmt.Errorf("get enabled addresses for device: %w", err)
	}

	return addresses, nil
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

	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		for i, addressID := range addressIDs {
			disabledAddress, err := r.recordAddressEvent(ctx, addressID, false, source)
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

// RefreshAddress records activity for an already-enabled address
func (r *Repository) RefreshAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error) {
	return r.EnableAddress(ctx, addressID, source)
}

// insertAddressEvent records an event in the address_events audit table without modifying the address itself.
func (r *Repository) insertAddressEvent(ctx context.Context, addressID AddressID, isEnabled bool, source EventSource, at time.Time) error {
	query := `
		INSERT INTO address_events (address_id, is_enabled, source, created_at)
		VALUES (?, ?, ?, ?)
	`

	if _, err := r.db.ExecContext(ctx, query, addressID, isEnabled, source, at); err != nil {
		return fmt.Errorf("failed to record event: %w", err)
	}
	return nil
}

func (r *Repository) recordAddressEvent(ctx context.Context, addressID AddressID, isEnabled bool, source EventSource) (*Address, error) {
	var finalAddress *Address
	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		now := time.Now().UTC()

		if err := r.insertAddressEvent(ctx, addressID, isEnabled, source, now); err != nil {
			return err
		}

		updateState := `
		UPDATE addresses SET is_enabled = ?, source = ?, updated_at = ? WHERE id = ?
	`

		if _, err := r.db.ExecContext(ctx, updateState, isEnabled, source, now, addressID); err != nil {
			return fmt.Errorf("failed to update address state: %w", err)
		}

		var err error
		finalAddress, err = r.GetAddress(ctx, addressID)
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

// stateChangeFilter returns a SQL clause that keeps only state-change events
// (creation, enable↔disable transitions) by comparing each event's is_enabled
// with the immediately preceding event for the same address.
const stateChangeFilter = ` AND (
	NOT EXISTS (
		SELECT 1 FROM address_events prev
		WHERE prev.address_id = aev.address_id AND prev.id < aev.id
	)
	OR aev.is_enabled != (
		SELECT prev.is_enabled FROM address_events prev
		WHERE prev.address_id = aev.address_id AND prev.id < aev.id
		ORDER BY prev.id DESC LIMIT 1
	)
)`

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
	bucketArgs = append(bucketArgs, q.Granularity.StrftimeISO())
	bucketArgs = append(bucketArgs, baseArgs...)

	var buckets []AddressEventBucket
	if err := r.db.SelectContext(ctx, &buckets, bucketsQuery, bucketArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("get history buckets: %w", err)
	}
	if buckets == nil {
		buckets = []AddressEventBucket{}
	}

	// ── Events (paginated) ───────────────────────────────────────────────
	// When IncludeAll is false, append a correlated subquery that keeps only
	// state-change events (first event per address, or is_enabled differs from
	// the immediately preceding event for the same address).
	var scFilter string
	if !q.IncludeAll {
		scFilter = stateChangeFilter
	}

	// Count total (without cursor)
	countQuery := `
		SELECT COUNT(*)
		FROM address_events aev
		JOIN addresses a ON a.id = aev.address_id
		JOIN devices d ON d.id = a.device_id
	` + joinWhere(filters) + scFilter

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
	` + joinWhere(eventFilters) + scFilter + `
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
