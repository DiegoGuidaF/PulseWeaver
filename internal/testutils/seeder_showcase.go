//go:build test

package testutils

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

// SeedShowcaseWorld returns a Seeder pre-loaded with a presentable, demo-quality
// dataset: a small remote team self-hosting recognizable services (Jellyfin,
// Nextcloud, Gitea, Vaultwarden, Grafana, Proxmox, Home Assistant, …). It is the
// counterpart to SeedFullWorld — same builder, presentable fixtures — intended
// for screenshots, walkthroughs, and demos rather than entity-count assertions.
//
// The login stays admin / TestAdminPassword (the bootstrap admin renders as
// "Administrator", a bypass user). The generated traffic spans the last 24h with
// a diurnal curve so the dashboard's raw-window widgets show real shapes:
// per-entity attribution, deny-reason split, a deny-rate world map (legitimate
// team traffic from US/GB/ES/IE plus a CN/RU/NL-dominated wall of denied scanner
// noise), top-denied IPs, and three pending host suggestions.
//
// Posture distribution: 1 bypass (admin), 4 live-with-access (Sarah, James,
// Maria, Liam), 1 live-no-host-access (Noah — live device, no grants), 1
// no-live-ips (Priya — stale device), 1 no-access (Tom — invited, no device).
func SeedShowcaseWorld(t *testing.T) *Seeder {
	t.Helper()

	const day = 24 * time.Hour
	leaseSarah := time.Now().Add(27 * day)
	leaseJames := time.Now().Add(30 * day)

	s := NewSeeder(t)

	// Groups → services.
	s.WithGroup(GroupFixture{Name: "Media", Color: "#7950F2", Icon: "IconDeviceTv"}).
		WithGroup(GroupFixture{Name: "Productivity", Color: "#4C6EF5", Icon: "IconCode"}).
		WithGroup(GroupFixture{Name: "Infrastructure", Color: "#E8590C", Icon: "IconServer2"}).
		WithGroup(GroupFixture{Name: "Smart Home", Color: "#0CA678", Icon: "IconHome"})

	for fqdn, group := range showcaseHostGroup {
		s.WithHost(HostFixture{FQDN: fqdn, Groups: []string{group}})
	}

	// Users. Sarah is a login-ready admin; the rest are plain accounts.
	s.WithUser(UserFixture{Name: "sarah_chen", Role: auth.AdminRole, DisplayName: "Sarah Chen", Email: "sarah.chen@example.com"}).
		WithUser(UserFixture{Name: "james_wilson", DisplayName: "James Wilson", Email: "james.wilson@example.com"}).
		WithUser(UserFixture{Name: "maria_garcia", DisplayName: "Maria Garcia", Email: "maria.garcia@example.com"}).
		WithUser(UserFixture{Name: "liam_murphy", DisplayName: "Liam Murphy", Email: "liam.murphy@example.com"}).
		WithUser(UserFixture{Name: "priya_patel", DisplayName: "Priya Patel", Email: "priya.patel@example.com"}).
		WithUser(UserFixture{Name: "noah_kim", DisplayName: "Noah Kim", Email: "noah.kim@example.com"}).
		WithUser(UserFixture{Name: "tom_becker", DisplayName: "Tom Becker", Email: "tom.becker@example.com"})

	// Host-group grants. The bootstrap admin bypasses; Noah and Tom get nothing.
	s.SetUserAccess(auth.BootstrapAdminUsername, true).
		SetUserAccess("sarah_chen", false, "Media", "Productivity", "Infrastructure", "Smart Home").
		SetUserAccess("james_wilson", false, "Media", "Productivity").
		SetUserAccess("maria_garcia", false, "Media", "Smart Home").
		SetUserAccess("liam_murphy", false, "Productivity").
		SetUserAccess("priya_patel", false, "Media")

	// Network policies (CIDR grants). Home LAN bypasses the host check.
	s.WithPolicy(PolicyFixture{Name: "Home LAN", CIDR: "192.168.1.0/24", Desc: "Trusted home network — unrestricted"}).
		WithPolicy(PolicyFixture{Name: "WireGuard VPN", CIDR: "10.8.0.0/24", Desc: "Remote-access tunnel for staff"}).
		WithPolicy(PolicyFixture{Name: "Office Network", CIDR: "198.51.100.0/24", Desc: "Branch office subnet"}).
		WithPolicy(PolicyFixture{Name: "Tailscale Mesh", CIDR: "100.64.0.0/10", Desc: "Personal mesh devices"}).
		WithPolicyBypassHostCheck("Home LAN").
		AssignGroupsToPolicy("WireGuard VPN", "Productivity", "Infrastructure").
		AssignGroupsToPolicy("Office Network", "Media", "Productivity").
		AssignGroupsToPolicy("Tailscale Mesh", "Productivity")

	// Devices (one or two per active user; Tom has none).
	for _, d := range showcaseDevices {
		s.WithDevice(DeviceFixture{Name: d.name, OwnerUser: d.owner, Icon: d.icon, GenerateAPIKey: true})
		s.WithPairing(PairingFixture{Device: d.name, Status: "used"})
	}
	s.WithDeviceLeaseRule(DeviceLeaseRuleFixture{Device: "Sarah's MacBook Pro", TTLSeconds: 27 * 24 * 3600}).
		WithDeviceLeaseRule(DeviceLeaseRuleFixture{Device: "James's Desktop", TTLSeconds: 30 * 24 * 3600})

	// Addresses: each live device reports its residential IP via heartbeat; Sarah
	// keeps one stale prior IP; Priya's device is offline (disabled).
	s.WithAddress(AddressFixture{Device: "Sarah's MacBook Pro", IP: "73.92.140.7", Source: device.EventSourceHeartbeat, ExpiresAt: &leaseSarah}).
		WithAddress(AddressFixture{Device: "Sarah's MacBook Pro", IP: "68.34.201.18", Source: device.EventSourceHeartbeat, Disabled: true}).
		WithAddress(AddressFixture{Device: "Sarah's iPhone", IP: "98.207.55.33", Source: device.EventSourceHeartbeat}).
		WithAddress(AddressFixture{Device: "James's Desktop", IP: "86.180.44.21", Source: device.EventSourceHeartbeat, ExpiresAt: &leaseJames}).
		WithAddress(AddressFixture{Device: "James's Pixel", IP: "86.14.220.9", Source: device.EventSourceHeartbeat}).
		WithAddress(AddressFixture{Device: "Maria's Laptop", IP: "88.6.120.40", Source: device.EventSourceHeartbeat}).
		WithAddress(AddressFixture{Device: "Maria's iPhone", IP: "83.36.77.18", Source: device.EventSourceHeartbeat}).
		WithAddress(AddressFixture{Device: "Liam's ThinkPad", IP: "86.43.220.11", Source: device.EventSourceHeartbeat}).
		WithAddress(AddressFixture{Device: "Priya's iPhone", IP: "49.36.220.7", Source: device.EventSourceHeartbeat, Disabled: true}).
		WithAddress(AddressFixture{Device: "Noah's Laptop", IP: "91.62.40.10", Source: device.EventSourceHeartbeat}).
		WithAddress(AddressFixture{Device: "Admin Workstation", IP: "24.6.50.10", Source: device.EventSourceHeartbeat})

	s.WithPolicyInitialize()
	s.WithGeneratedTraffic(showcaseTraffic())
	return s
}

// ── showcase data ─────────────────────────────────────────────────────────────

var (
	showcaseMedia        = []string{"jellyfin.example.com", "plex.example.com", "immich.example.com"}
	showcaseProductivity = []string{"nextcloud.example.com", "git.example.com", "vault.example.com", "paperless.example.com"}
	showcaseInfra        = []string{"proxmox.example.com", "grafana.example.com", "pihole.example.com"}
	showcaseSmartHome    = []string{"home-assistant.example.com", "frigate.example.com"}
)

// showcaseHostGroup maps every known host to its single owning group.
var showcaseHostGroup = func() map[string]string {
	m := map[string]string{}
	for _, h := range showcaseMedia {
		m[h] = "Media"
	}
	for _, h := range showcaseProductivity {
		m[h] = "Productivity"
	}
	for _, h := range showcaseInfra {
		m[h] = "Infrastructure"
	}
	for _, h := range showcaseSmartHome {
		m[h] = "Smart Home"
	}
	return m
}()

func showcaseAllHosts() []string {
	out := append([]string{}, showcaseMedia...)
	out = append(out, showcaseProductivity...)
	out = append(out, showcaseInfra...)
	return append(out, showcaseSmartHome...)
}

var showcaseDevices = []struct{ name, owner, icon string }{
	{"Sarah's MacBook Pro", "sarah_chen", "💻"},
	{"Sarah's iPhone", "sarah_chen", "📱"},
	{"James's Desktop", "james_wilson", "🖥️"},
	{"James's Pixel", "james_wilson", "📱"},
	{"Maria's Laptop", "maria_garcia", "💻"},
	{"Maria's iPhone", "maria_garcia", "📱"},
	{"Liam's ThinkPad", "liam_murphy", "💻"},
	{"Priya's iPhone", "priya_patel", "📱"},
	{"Noah's Laptop", "noah_kim", "💻"},
	{"Admin Workstation", auth.BootstrapAdminUsername, "🖥️"},
}

// showcaseGeo holds the country/ASN each public IP resolves to in the bundled
// DB-IP databases, so the stored country map and the live top-denied lookups agree.
var showcaseGeo = map[string]geoip.Result{
	"73.92.140.7":    {CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 7922, ASNOrg: "Comcast Cable Communications, LLC"},
	"98.207.55.33":   {CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 7922, ASNOrg: "Comcast Cable Communications, LLC"},
	"24.6.50.10":     {CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 7922, ASNOrg: "Comcast Cable Communications, LLC"},
	"86.180.44.21":   {CountryCode: "GB", CountryName: "United Kingdom", ContinentCode: "EU", ASN: 2856, ASNOrg: "British Telecommunications PLC"},
	"86.14.220.9":    {CountryCode: "GB", CountryName: "United Kingdom", ContinentCode: "EU", ASN: 5089, ASNOrg: "Virgin Media Limited"},
	"88.6.120.40":    {CountryCode: "ES", CountryName: "Spain", ContinentCode: "EU", ASN: 3352, ASNOrg: "TELEFONICA DE ESPANA S.A.U."},
	"83.36.77.18":    {CountryCode: "ES", CountryName: "Spain", ContinentCode: "EU", ASN: 3352, ASNOrg: "TELEFONICA DE ESPANA S.A.U."},
	"86.43.220.11":   {CountryCode: "IE", CountryName: "Ireland", ContinentCode: "EU", ASN: 5466, ASNOrg: "Eircom Limited"},
	"91.62.40.10":    {CountryCode: "DE", CountryName: "Germany", ContinentCode: "EU", ASN: 3320, ASNOrg: "Deutsche Telekom AG"},
	"218.92.0.118":   {CountryCode: "CN", CountryName: "China", ContinentCode: "AS", ASN: 4134, ASNOrg: "Chinanet"},
	"116.31.116.40":  {CountryCode: "CN", CountryName: "China", ContinentCode: "AS", ASN: 4134, ASNOrg: "Chinanet"},
	"61.177.172.10":  {CountryCode: "CN", CountryName: "China", ContinentCode: "AS", ASN: 4134, ASNOrg: "Chinanet"},
	"221.181.10.5":   {CountryCode: "CN", CountryName: "China", ContinentCode: "AS", ASN: 9808, ASNOrg: "China Mobile"},
	"45.155.205.50":  {CountryCode: "RU", CountryName: "Russia", ContinentCode: "EU", ASN: 208677, ASNOrg: "Cloud Technologies LLC"},
	"5.188.206.10":   {CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 200391, ASNOrg: "KREZ 999 EOOD"},
	"80.82.77.139":   {CountryCode: "NL", CountryName: "The Netherlands", ContinentCode: "EU", ASN: 202425, ASNOrg: "IP Volume inc"},
	"89.248.165.50":  {CountryCode: "NL", CountryName: "The Netherlands", ContinentCode: "EU", ASN: 202425, ASNOrg: "IP Volume inc"},
	"193.32.162.40":  {CountryCode: "NL", CountryName: "The Netherlands", ContinentCode: "EU", ASN: 47890, ASNOrg: "UNMANAGED LTD"},
	"185.220.101.40": {CountryCode: "DE", CountryName: "Germany", ContinentCode: "EU", ASN: 60729, ASNOrg: "Stiftung Erneuerbare Freiheit"},
	"123.30.100.20":  {CountryCode: "VN", CountryName: "Vietnam", ContinentCode: "AS", ASN: 45899, ASNOrg: "VNPT Corp"},
	"14.177.10.5":    {CountryCode: "VN", CountryName: "Vietnam", ContinentCode: "AS", ASN: 45899, ASNOrg: "VNPT Corp"},
	"103.102.40.5":   {CountryCode: "PK", CountryName: "Pakistan", ContinentCode: "AS", ASN: 58895, ASNOrg: "E Bone Network (Pvt.) Limited"},
	"200.160.2.3":    {CountryCode: "BR", CountryName: "Brazil", ContinentCode: "SA", ASN: 22548, ASNOrg: "NIC.BR"},
	"177.71.207.10":  {CountryCode: "BR", CountryName: "Brazil", ContinentCode: "SA", ASN: 16509, ASNOrg: "Amazon.com, Inc."},
	"41.79.10.5":     {CountryCode: "KE", CountryName: "Kenya", ContinentCode: "AF", ASN: 37305, ASNOrg: "Frontier Optical Networks Ltd"},
	"196.30.100.7":   {CountryCode: "ZA", CountryName: "South Africa", ContinentCode: "AF", ASN: 16637, ASNOrg: "MTN SA"},
	"79.140.10.20":   {CountryCode: "UA", CountryName: "Ukraine", ContinentCode: "EU", ASN: 6876, ASNOrg: "TENET LLC"},
	"194.165.16.10":  {CountryCode: "LT", CountryName: "Lithuania", ContinentCode: "EU", ASN: 48721, ASNOrg: "Flyservers S.A."},
}

func showcaseGeoFor(ip string) *geoip.Result {
	if g, ok := showcaseGeo[ip]; ok {
		return &g
	}
	return nil
}

// showcaseTraffic builds the weighted, time-spread traffic profile: legitimate
// team + network-policy traffic, configured users denied at hosts they lack, and
// a wall of denied internet-scanner noise — plus a few unknown hosts that surface
// as suggestions.
func showcaseTraffic() TrafficProfile {
	methods := []string{"GET", "GET", "GET", "POST"}
	uris := []string{"/", "/api/health", "/web/index.html", "/dashboard", "/library", "/api/v1/status"}
	hostDeny := new(policy.DenyReasonHostNotAllowed)
	ipDeny := new(policy.DenyReasonIPNotRegistered)

	allHosts := showcaseAllHosts()
	mediaProd := append(append([]string{}, showcaseMedia...), showcaseProductivity...)
	prodInfra := append(append([]string{}, showcaseProductivity...), showcaseInfra...)
	infraSmart := append(append([]string{}, showcaseInfra...), showcaseSmartHome...)
	mediaInfra := append(append([]string{}, showcaseMedia...), showcaseInfra...)

	var streams []TrafficStream

	// Per-device team traffic: allowed at granted hosts, denied at the rest.
	devTraffic := []struct {
		device, ip            string
		allowHosts, denyHosts []string
		allowN, denyN         int
	}{
		{"Sarah's MacBook Pro", "73.92.140.7", allHosts, nil, 260, 0},
		{"Sarah's iPhone", "98.207.55.33", allHosts, nil, 260, 0},
		{"Admin Workstation", "24.6.50.10", allHosts, nil, 210, 0},
		{"James's Desktop", "86.180.44.21", mediaProd, infraSmart, 180, 22},
		{"James's Pixel", "86.14.220.9", mediaProd, infraSmart, 170, 18},
		{"Maria's Laptop", "88.6.120.40", append(append([]string{}, showcaseMedia...), showcaseSmartHome...), prodInfra, 150, 20},
		{"Maria's iPhone", "83.36.77.18", append(append([]string{}, showcaseMedia...), showcaseSmartHome...), prodInfra, 130, 15},
		{"Liam's ThinkPad", "86.43.220.11", showcaseProductivity, mediaInfra, 150, 25},
		{"Noah's Laptop", "91.62.40.10", nil, allHosts, 0, 150}, // no grants → all denied
	}
	for _, d := range devTraffic {
		geo := showcaseGeoFor(d.ip)
		if d.allowN > 0 {
			streams = append(streams, TrafficStream{
				Count: d.allowN, ClientIPs: []string{d.ip}, Outcome: true,
				Hosts: d.allowHosts, Devices: []string{d.device}, Geo: geo,
				Methods: methods, URIs: uris,
			})
		}
		if d.denyN > 0 {
			streams = append(streams, TrafficStream{
				Count: d.denyN, ClientIPs: []string{d.ip}, Outcome: false, DenyReason: hostDeny,
				Hosts: d.denyHosts, Devices: []string{d.device}, Geo: geo,
				Methods: methods, URIs: uris,
			})
		}
	}

	// Network-policy traffic: allowed via CIDR grants (private IPs → no geo).
	policyTraffic := []struct {
		name    string
		clients []string
		hosts   []string
		n       int
	}{
		{"Home LAN", []string{"192.168.1.10", "192.168.1.20", "192.168.1.42", "192.168.1.50"}, allHosts, 260},
		{"WireGuard VPN", []string{"10.8.0.2", "10.8.0.5", "10.8.0.9"}, prodInfra, 180},
		{"Office Network", []string{"198.51.100.20", "198.51.100.21", "198.51.100.22"}, mediaProd, 150},
		{"Tailscale Mesh", []string{"100.64.1.5", "100.64.1.8"}, showcaseProductivity, 90},
	}
	for _, p := range policyTraffic {
		streams = append(streams, TrafficStream{
			Count: p.n, ClientIPs: p.clients, Outcome: true,
			Hosts: p.hosts, PolicyName: p.name, Methods: methods, URIs: uris,
		})
	}

	// Internet-scanner noise: denied unknown IPs probing common attack targets.
	attackTargets := []string{"vault.example.com", "git.example.com", "grafana.example.com", "proxmox.example.com", "nextcloud.example.com", "pihole.example.com"}
	scanURIs := []string{"/", "/admin", "/api/login", "/wp-login.php", "/.env", "/.git/config"}
	scanMethods := []string{"GET", "POST", "HEAD"}
	scanners := []struct {
		ip string
		n  int
	}{
		{"218.92.0.118", 232}, {"116.31.116.40", 188}, {"45.155.205.50", 151},
		{"5.188.206.10", 130}, {"185.220.101.40", 111}, {"80.82.77.139", 95},
		{"89.248.165.50", 76}, {"61.177.172.10", 74}, {"193.32.162.40", 59},
		{"123.30.100.20", 50}, {"14.177.10.5", 44}, {"103.102.40.5", 40},
		{"200.160.2.3", 35}, {"177.71.207.10", 30}, {"221.181.10.5", 28},
		{"41.79.10.5", 22}, {"196.30.100.7", 18}, {"79.140.10.20", 16},
		{"194.165.16.10", 12},
	}
	for _, sc := range scanners {
		streams = append(streams, TrafficStream{
			Count: sc.n, ClientIPs: []string{sc.ip}, Outcome: false, DenyReason: ipDeny,
			Hosts: attackTargets, Geo: showcaseGeoFor(sc.ip), Methods: scanMethods, URIs: scanURIs,
		})
	}

	// Unknown-but-real services seen on the LAN → pending host suggestions.
	for _, sug := range []struct {
		fqdn string
		n    int
	}{{"photoprism.example.com", 8}, {"uptime-kuma.example.com", 7}, {"audiobookshelf.example.com", 8}} {
		streams = append(streams, TrafficStream{
			Count: sug.n, ClientIPs: []string{"192.168.1.10"}, Outcome: true,
			Hosts: []string{sug.fqdn}, PolicyName: "Home LAN", Methods: methods, URIs: uris,
		})
	}

	return TrafficProfile{Window: 24 * time.Hour, Diurnal: true, Seed: 42, Streams: streams}
}
