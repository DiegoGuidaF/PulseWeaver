package geoip

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/oschwald/geoip2-golang"
)

type mmdbReaders struct {
	countryDB *geoip2.Reader
	asnDB     *geoip2.Reader
}

// Lookup wraps MMDB readers. A zero-value Lookup is valid — all resolves return empty Result.
// Concurrency-safe: atomic.Pointer for lock-free reads during Reload.
type Lookup struct {
	readers atomic.Pointer[mmdbReaders]
	dataDir string
}

// New initializes GeoIP enrichment from conf. Returns a zero-value Lookup
// (enrichment disabled, all resolves return empty Result) when enrichment is
// disabled or the DB-IP download fails (fail-open).
func New(ctx context.Context, conf config.ConfGeoIP, logger *slog.Logger) (*Lookup, error) {
	if !conf.Enabled {
		logger.Info("geoip: enrichment disabled (GEOIP_ENABLED=false)")
		return &Lookup{}, nil
	}

	cp, ap, err := SyncDBIPFiles(ctx, conf.DataDir, logger)
	if err != nil {
		logger.Error("geoip: DB-IP download failed, enrichment disabled — set GEOIP_ENABLED=false to suppress",
			slog.Any(logging.AttrKeyError, err))
		return &Lookup{}, nil // fail-open: no readers, all resolves return empty
	}

	l := &Lookup{dataDir: conf.DataDir}
	if err := l.Reload(cp, ap); err != nil {
		return nil, fmt.Errorf("geoip open: %w", err)
	}
	return l, nil
}

// Close closes underlying readers. Nil-safe.
func (l *Lookup) Close() error {
	if l == nil {
		return nil
	}
	r := l.readers.Swap(nil)
	if r == nil {
		return nil
	}
	closeReaders(r)
	return nil
}

// Resolve resolves ip to a Result. Fail-open: any error returns empty Result.
// Nil-safe: a nil receiver returns empty Result.
func (l *Lookup) Resolve(ip string) Result {
	if l == nil {
		return Result{}
	}
	r := l.readers.Load()
	if r == nil {
		return Result{}
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return Result{}
	}
	var res Result
	if record, err := r.countryDB.Country(parsed); err == nil {
		res.CountryCode = record.Country.IsoCode
		res.CountryName = record.Country.Names["en"]
		res.ContinentCode = record.Continent.Code
	}
	if record, err := r.asnDB.ASN(parsed); err == nil {
		res.ASN = uint(record.AutonomousSystemNumber)
		res.ASNOrg = record.AutonomousSystemOrganization
	}
	return res
}

// Reload opens new readers, swaps the atomic pointer, and closes old ones.
// On failure it closes any partially opened readers and returns an error,
// leaving the old readers active.
func (l *Lookup) Reload(countryPath, asnPath string) error {
	country, err := geoip2.Open(countryPath)
	if err != nil {
		return err
	}
	asn, err := geoip2.Open(asnPath)
	if err != nil {
		_ = country.Close()
		return err
	}
	old := l.readers.Swap(&mmdbReaders{countryDB: country, asnDB: asn})
	if old != nil {
		closeReaders(old)
	}
	return nil
}

func closeReaders(r *mmdbReaders) {
	_ = r.countryDB.Close()
	_ = r.asnDB.Close()
}
