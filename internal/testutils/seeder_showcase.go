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
	s.WithGroup(GroupMedia).
		WithGroup(GroupProductivity).
		WithGroup(GroupInfrastructure).
		WithGroup(GroupSmartHome)

	for fqdn, group := range showcaseHostGroup {
		s.WithHost(HostFixture{FQDN: fqdn, Groups: []string{group}})
	}

	// Users. Sarah is a login-ready admin; the rest are plain accounts.
	s.WithUser(PersonSarah).
		WithUser(PersonJames).
		WithUser(PersonMaria).
		WithUser(PersonLiam).
		WithUser(PersonPriya).
		WithUser(PersonNoah).
		WithUser(PersonTom)

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
	showcaseMedia        = ServiceMediaHosts
	showcaseProductivity = ServiceProductivityHosts
	showcaseInfra        = ServiceInfraHosts
	showcaseSmartHome    = ServiceSmartHomeHosts
)

// showcaseHostGroup maps every known host to its single owning group.
var showcaseHostGroup = func() map[string]string {
	m := map[string]string{}
	for _, h := range showcaseMedia {
		m[h] = GroupMedia.Name
	}
	for _, h := range showcaseProductivity {
		m[h] = GroupProductivity.Name
	}
	for _, h := range showcaseInfra {
		m[h] = GroupInfrastructure.Name
	}
	for _, h := range showcaseSmartHome {
		m[h] = GroupSmartHome.Name
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
	"73.92.140.7":    GeoResult("US", 7922, "Comcast Cable Communications, LLC"),
	"98.207.55.33":   GeoResult("US", 7922, "Comcast Cable Communications, LLC"),
	"24.6.50.10":     GeoResult("US", 7922, "Comcast Cable Communications, LLC"),
	"86.180.44.21":   GeoResult("GB", 2856, "British Telecommunications PLC"),
	"86.14.220.9":    GeoResult("GB", 5089, "Virgin Media Limited"),
	"88.6.120.40":    GeoResult("ES", 3352, "TELEFONICA DE ESPANA S.A.U."),
	"83.36.77.18":    GeoResult("ES", 3352, "TELEFONICA DE ESPANA S.A.U."),
	"86.43.220.11":   GeoResult("IE", 5466, "Eircom Limited"),
	"91.62.40.10":    GeoResult("DE", 3320, "Deutsche Telekom AG"),
	"218.92.0.118":   GeoResult("CN", 4134, "Chinanet"),
	"116.31.116.40":  GeoResult("CN", 4134, "Chinanet"),
	"61.177.172.10":  GeoResult("CN", 4134, "Chinanet"),
	"221.181.10.5":   GeoResult("CN", 9808, "China Mobile"),
	"45.155.205.50":  GeoResult("RU", 208677, "Cloud Technologies LLC"),
	"5.188.206.10":   GeoResult("US", 200391, "KREZ 999 EOOD"),
	"80.82.77.139":   GeoResult("NL", 202425, "IP Volume inc"),
	"89.248.165.50":  GeoResult("NL", 202425, "IP Volume inc"),
	"193.32.162.40":  GeoResult("NL", 47890, "UNMANAGED LTD"),
	"185.220.101.40": GeoResult("DE", 60729, "Stiftung Erneuerbare Freiheit"),
	"123.30.100.20":  GeoResult("VN", 45899, "VNPT Corp"),
	"14.177.10.5":    GeoResult("VN", 45899, "VNPT Corp"),
	"103.102.40.5":   GeoResult("PK", 58895, "E Bone Network (Pvt.) Limited"),
	"200.160.2.3":    GeoResult("BR", 22548, "NIC.BR"),
	"177.71.207.10":  GeoResult("BR", 16509, "Amazon.com, Inc."),
	"41.79.10.5":     GeoResult("KE", 37305, "Frontier Optical Networks Ltd"),
	"196.30.100.7":   GeoResult("ZA", 16637, "MTN SA"),
	"79.140.10.20":   GeoResult("UA", 6876, "TENET LLC"),
	"194.165.16.10":  GeoResult("LT", 48721, "Flyservers S.A."),
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
