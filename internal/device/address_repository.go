package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// DeleteAddressEventsOlderThan removes address_events rows with created_at before the given time.
// Returns the number of rows deleted.
func (r *Repository) DeleteAddressEventsOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM address_events WHERE created_at < ?`, before)
	if err != nil {
		return 0, fmt.Errorf("delete address_events older than %s: %w", before.Format(time.RFC3339), err)
	}
	return result.RowsAffected()
}

func (r *Repository) GetAddress(ctx context.Context, addressID ids.AddressID) (*Address, error) {
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

func (r *Repository) CreateAddress(ctx context.Context, params CreateAddressParams, source EventSource) (*Address, error) {
	var address *Address

	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		now := time.Now().UTC()

		query := `
		INSERT INTO addresses (device_id, ip, is_enabled, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id
	`
		var addressID ids.AddressID
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

func (r *Repository) GetAddressForDeviceByIP(ctx context.Context, deviceID ids.DeviceID, ip netip.Addr) (*Address, error) {
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

func (r *Repository) CheckAddressOwnership(ctx context.Context, deviceID ids.DeviceID, addressID ids.AddressID) error {
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

func (r *Repository) DisableAddress(ctx context.Context, addressID ids.AddressID) (*Address, error) {
	return r.recordAddressEvent(ctx, addressID, false, EventSourceManual)
}

func (r *Repository) DisableAddresses(ctx context.Context, addressIDs []ids.AddressID, source EventSource) ([]Address, error) {
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

func (r *Repository) EnableAddress(ctx context.Context, addressID ids.AddressID, source EventSource) (*Address, error) {
	return r.recordAddressEvent(ctx, addressID, true, source)
}

// RefreshAddress records activity for an already-enabled address
func (r *Repository) RefreshAddress(ctx context.Context, addressID ids.AddressID, source EventSource) (*Address, error) {
	return r.EnableAddress(ctx, addressID, source)
}

func (r *Repository) GetEnabledIPEntries(ctx context.Context) ([]IPEntry, error) {
	var entries []IPEntry

	// Returns ALL enabled rows (multiple per IP when devices share an address).
	// The policy layer groups by IP and applies deny-wins intersection.
	query := `
		SELECT a.ip, a.device_id, a.id AS address_id, d.owner_id AS user_id
		FROM addresses a
		JOIN devices d ON d.id = a.device_id
		WHERE a.is_enabled = 1
		AND d.deleted_at is NULL
		ORDER BY a.updated_at DESC
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
func (r *Repository) GetEnabledAddressesForDevice(ctx context.Context, deviceID ids.DeviceID) ([]Address, error) {
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

// insertAddressEvent records an event in the address_events audit table without modifying the address itself.
func (r *Repository) insertAddressEvent(ctx context.Context, addressID ids.AddressID, isEnabled bool, source EventSource, at time.Time) error {
	query := `
		INSERT INTO address_events (address_id, is_enabled, source, created_at)
		VALUES (?, ?, ?, ?)
	`

	if _, err := r.db.ExecContext(ctx, query, addressID, isEnabled, source, at); err != nil {
		return fmt.Errorf("failed to record event: %w", err)
	}
	return nil
}

func (r *Repository) recordAddressEvent(ctx context.Context, addressID ids.AddressID, isEnabled bool, source EventSource) (*Address, error) {
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

// addressHistoryFilters builds the shared WHERE conditions for the buckets, count, and
// events queries so the three can never drift. Column references are fixed constants;
// callers supply only values, which squirrel parameterises. A device-ID slice expands
// to an IN clause (empty slice is omitted).
func addressHistoryFilters(q AddressHistoryQuery) sq.And {
	cond := sq.And{sq.Expr("d.deleted_at IS NULL")}

	if len(q.DeviceIDs) > 0 {
		cond = append(cond, sq.Eq{"a.device_id": q.DeviceIDs})
	}
	if !q.From.IsZero() {
		cond = append(cond, sq.GtOrEq{"aev.created_at": q.From})
	}
	if !q.To.IsZero() {
		cond = append(cond, sq.LtOrEq{"aev.created_at": q.To})
	}
	if q.Source != nil {
		cond = append(cond, sq.Eq{"aev.source": *q.Source})
	}
	if q.IsEnabled != nil {
		cond = append(cond, sq.Eq{"aev.is_enabled": *q.IsEnabled})
	}
	if q.IP != nil {
		cond = append(cond, sq.Expr(`a.ip LIKE ? ESCAPE '\'`, "%"+database.EscapeLIKE(*q.IP)+"%"))
	}

	return cond
}

// stateChangeCond keeps only state-change events (creation, enable↔disable
// transitions) by comparing each event's is_enabled with the immediately preceding
// event for the same address. Added as a WHERE condition when IncludeAll is false.
const stateChangeCond = `(
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

func (r *Repository) GetAddressHistory(ctx context.Context, q AddressHistoryQuery) (AddressHistory, error) {
	cond := addressHistoryFilters(q)

	// ── Buckets ──────────────────────────────────────────────────────────
	// active_count = addresses whose last event in the bucket was is_enabled=1.
	//   "Last event" = no later event exists for the same address in the same bucket.
	//   Detected via NOT EXISTS on address_events directly (avoids CTE self-ref).
	// gap_count = addresses that had any is_enabled=0 (expiry) event in the bucket.
	//
	// The strftime format string is a column-level placeholder, so squirrel emits its
	// args before the WHERE args automatically — no manual arg layout required.
	bucketFmt := q.Granularity.StrftimeISO()
	bucketsSQL, bucketArgs, err := sq.
		Select().
		Column("strftime(?, aev.created_at) AS bucket", bucketFmt).
		Column(`COUNT(DISTINCT CASE
			WHEN aev.is_enabled = 1
			 AND NOT EXISTS (
				 SELECT 1 FROM address_events later
				 WHERE later.address_id = aev.address_id
				   AND later.id > aev.id
				   AND strftime(?, later.created_at) = strftime(?, aev.created_at)
			 )
			THEN aev.address_id
		END) AS active_count`, bucketFmt, bucketFmt).
		Column("COUNT(DISTINCT CASE WHEN aev.is_enabled = 0 THEN aev.address_id END) AS gap_count").
		Column("COUNT(*) AS event_count").
		From("address_events aev").
		Join("addresses a ON a.id = aev.address_id").
		Join("devices d ON d.id = a.device_id").
		Where(cond).
		GroupBy("bucket").
		OrderBy("bucket ASC").
		ToSql()
	if err != nil {
		return AddressHistory{}, fmt.Errorf("build history buckets query: %w", err)
	}

	var buckets []AddressEventBucket
	if err := r.db.SelectContext(ctx, &buckets, bucketsSQL, bucketArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("get history buckets: %w", err)
	}
	if buckets == nil {
		buckets = []AddressEventBucket{}
	}

	// ── Events (paginated) ───────────────────────────────────────────────
	// Shared base for the count and the page query. When IncludeAll is false, restrict
	// to state-change events (first event per address, or is_enabled differs from the
	// immediately preceding event for the same address).
	base := sq.
		Select().
		From("address_events aev").
		Join("addresses a ON a.id = aev.address_id").
		Join("devices d ON d.id = a.device_id").
		Where(cond)
	if !q.IncludeAll {
		base = base.Where(sq.Expr(stateChangeCond))
	}

	// Count (without cursor).
	countSQL, countArgs, err := base.Column("COUNT(*)").ToSql()
	if err != nil {
		return AddressHistory{}, fmt.Errorf("build history count query: %w", err)
	}
	var totalEvents int
	if err := r.db.GetContext(ctx, &totalEvents, countSQL, countArgs...); err != nil {
		return AddressHistory{}, fmt.Errorf("count history events: %w", err)
	}

	// Events page (cursor + limit).
	eventsB := base.Columns(
		"aev.id", "aev.created_at", "a.ip", "aev.is_enabled", "aev.source",
		"a.device_id", "d.name AS device_name",
	)
	if q.BeforeID != nil {
		eventsB = eventsB.Where(sq.Lt{"aev.id": *q.BeforeID})
	}
	eventsB = eventsB.OrderBy("aev.id DESC").Limit(uint64(q.Limit))

	eventsSQL, eventArgs, err := eventsB.ToSql()
	if err != nil {
		return AddressHistory{}, fmt.Errorf("build history events query: %w", err)
	}

	var events []AddressStateChange
	if err := r.db.SelectContext(ctx, &events, eventsSQL, eventArgs...); err != nil {
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
