//go:build test

package testutils

import (
	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
)

// The shared identity layer: one realistic cast of people, service hosts, host
// groups, and country metadata that both seeded worlds compose from. Only the
// identity is shared — the per-world structure (which groups a user can reach,
// how many devices they own, their addresses, the access-log/traffic) is layered
// by SeedFullWorld and SeedShowcaseWorld separately, because those depend on
// addressing that the two worlds deliberately keep apart (private policy-test IPs
// vs public geolocated demo IPs).

// ── people roster ─────────────────────────────────────────────────────────────
//
// Identity only — username, display name, email, and role. Sarah is the single
// login-ready admin; everyone else is a plain account. Group access, devices and
// addresses are assigned per world.

var (
	PersonSarah = UserFixture{Name: "sarah_chen", Role: auth.AdminRole, DisplayName: "Sarah Chen", Email: "sarah.chen@example.com"}
	PersonJames = UserFixture{Name: "james_wilson", DisplayName: "James Wilson", Email: "james.wilson@example.com"}
	PersonMaria = UserFixture{Name: "maria_garcia", DisplayName: "Maria Garcia", Email: "maria.garcia@example.com"}
	PersonLiam  = UserFixture{Name: "liam_murphy", DisplayName: "Liam Murphy", Email: "liam.murphy@example.com"}
	PersonPriya = UserFixture{Name: "priya_patel", DisplayName: "Priya Patel", Email: "priya.patel@example.com"}
	PersonNoah  = UserFixture{Name: "noah_kim", DisplayName: "Noah Kim", Email: "noah.kim@example.com"}
	PersonTom   = UserFixture{Name: "tom_becker", DisplayName: "Tom Becker", Email: "tom.becker@example.com"}
)

// ── service groups ────────────────────────────────────────────────────────────
//
// The four self-hosting service families, with their display color and icon.

var (
	GroupMedia          = GroupFixture{Name: "Media", Color: "#7950F2", Icon: "IconDeviceTv"}
	GroupProductivity   = GroupFixture{Name: "Productivity", Color: "#4C6EF5", Icon: "IconCode"}
	GroupInfrastructure = GroupFixture{Name: "Infrastructure", Color: "#E8590C", Icon: "IconServer2"}
	GroupSmartHome      = GroupFixture{Name: "Smart Home", Color: "#0CA678", Icon: "IconHome"}
)

// ── service hosts ─────────────────────────────────────────────────────────────
//
// Recognizable self-hosted services, grouped by the family that owns them. A
// world declares whichever subset it needs; the FQDNs and grouping are the shared
// source of truth.

var (
	ServiceMediaHosts        = []string{"jellyfin.example.com", "plex.example.com", "immich.example.com"}
	ServiceProductivityHosts = []string{"nextcloud.example.com", "git.example.com", "vault.example.com", "paperless.example.com"}
	ServiceInfraHosts        = []string{"proxmox.example.com", "grafana.example.com", "pihole.example.com"}
	ServiceSmartHomeHosts    = []string{"home-assistant.example.com", "frigate.example.com"}
)

// ── GeoIP / country catalog ───────────────────────────────────────────────────
//
// One source of truth for country metadata (name + continent by ISO code), so the
// test fixtures and the showcase IP→geo map agree on every country and never
// repeat the name/continent literals. ASN + org stay per call because a single
// country is served by many networks.

var geoCountryCatalog = map[string]struct {
	name      string
	continent string
}{
	"US": {"United States", "NA"},
	"GB": {"United Kingdom", "EU"},
	"ES": {"Spain", "EU"},
	"IE": {"Ireland", "EU"},
	"DE": {"Germany", "EU"},
	"CN": {"China", "AS"},
	"RU": {"Russia", "EU"},
	"NL": {"The Netherlands", "EU"},
	"VN": {"Vietnam", "AS"},
	"PK": {"Pakistan", "AS"},
	"BR": {"Brazil", "SA"},
	"KE": {"Kenya", "AF"},
	"ZA": {"South Africa", "AF"},
	"UA": {"Ukraine", "EU"},
	"LT": {"Lithuania", "EU"},
}

// GeoResult builds a geoip.Result for an ISO country code, filling country name and
// continent from the shared catalog. It panics on an unknown code so a typo fails
// loudly at seed time rather than silently producing an empty country.
func GeoResult(countryCode string, asn uint, asnOrg string) geoip.Result {
	c, ok := geoCountryCatalog[countryCode]
	if !ok {
		panic("testutils.GeoResult: unknown country code " + countryCode)
	}
	return geoip.Result{
		CountryCode:   countryCode,
		CountryName:   c.name,
		ContinentCode: c.continent,
		ASN:           asn,
		ASNOrg:        asnOrg,
	}
}
