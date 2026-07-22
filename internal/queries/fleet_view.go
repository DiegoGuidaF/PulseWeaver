package queries

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/useraccess"
)

// The fleet readers, declared consumer-side (Go convention; precedent:
// GeoResolver). Each is one narrow query owned by the domain whose tables it
// reads. Only SubjectReader and FleetDeviceReader know owners exist; the rest
// are keyed by device id, which is what lets one composer serve both the fleet
// list and a single owner's detail view.
type (
	SubjectReader interface {
		SubjectRows(ctx context.Context, userID *ids.UserID) ([]useraccess.SubjectRow, error)
	}
	FleetDeviceReader interface {
		FleetRows(ctx context.Context, ownerID *ids.UserID) ([]device.FleetRow, error)
	}
	DeviceRulesReader interface {
		RulesForDevices(ctx context.Context, deviceIDs []ids.DeviceID) (map[ids.DeviceID][]httpapi.DeviceRuleSummary, error)
	}
	DevicePairingsReader interface {
		PairingsForDevices(ctx context.Context, deviceIDs []ids.DeviceID) (map[ids.DeviceID]httpapi.DevicePairingSummary, error)
	}
)

// FleetComposer assembles the owner→devices→rules/pairing tree from four
// batched reads. It holds no SQL of its own: a query written here means a
// reader is missing from a domain.
type FleetComposer struct {
	subjects SubjectReader
	devices  FleetDeviceReader
	rules    DeviceRulesReader
	pairings DevicePairingsReader
}

func NewFleetComposer(
	subjects SubjectReader,
	devices FleetDeviceReader,
	rules DeviceRulesReader,
	pairings DevicePairingsReader,
) *FleetComposer {
	return &FleetComposer{subjects: subjects, devices: devices, rules: rules, pairings: pairings}
}

// Fleet returns every owner with their devices, or just one owner's group when
// ownerID is set. An owner id that resolves to no user yields no groups at all,
// which is what lets the endpoint answer "unknown owner" without an existence
// probe or a 404 branch.
func (c *FleetComposer) Fleet(ctx context.Context, ownerID *ids.UserID) ([]httpapi.OwnerFleetGroup, error) {
	subjects, err := c.subjects.SubjectRows(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("compose fleet subjects: %w", err)
	}
	devices, err := c.devices.FleetRows(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("compose fleet devices: %w", err)
	}

	deviceIDs := make([]ids.DeviceID, len(devices))
	for i, row := range devices {
		deviceIDs[i] = ids.DeviceID(row.Device.Id)
	}

	rules, err := c.rules.RulesForDevices(ctx, deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("compose fleet rules: %w", err)
	}
	pairings, err := c.pairings.PairingsForDevices(ctx, deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("compose fleet pairings: %w", err)
	}

	return stitchFleet(subjects, devices, rules, pairings), nil
}

// stitchFleet groups devices under their owner and attaches each device's rules
// and pairing by id. Owner aggregates are summed from the device rows in hand:
// a second SQL implementation of the same count would drift from this one.
// Input ordering is the response ordering — subjects arrive sorted for display,
// devices by name.
func stitchFleet(
	subjects []useraccess.SubjectRow,
	devices []device.FleetRow,
	rules map[ids.DeviceID][]httpapi.DeviceRuleSummary,
	pairings map[ids.DeviceID]httpapi.DevicePairingSummary,
) []httpapi.OwnerFleetGroup {
	groups := make([]httpapi.OwnerFleetGroup, len(subjects))
	indexByOwner := make(map[ids.UserID]int, len(subjects))
	for i, subject := range subjects {
		groups[i] = httpapi.OwnerFleetGroup{
			Owner: httpapi.OwnerSummary{
				Id:              subject.ID.Int64(),
				DisplayName:     subject.DisplayName,
				Role:            subject.Role,
				BypassHostCheck: subject.BypassHostCheck,
				HostGroups:      subject.HostGroups,
			},
			Devices: []httpapi.FleetDevice{},
		}
		indexByOwner[subject.ID] = i
	}

	for _, row := range devices {
		idx, ok := indexByOwner[row.OwnerID]
		if !ok {
			continue
		}
		deviceID := ids.DeviceID(row.Device.Id)
		if deviceRules, ok := rules[deviceID]; ok {
			row.Device.Rules = deviceRules
		}
		if pairing, ok := pairings[deviceID]; ok {
			row.Device.Pairing = new(pairing)
		}
		group := &groups[idx]
		group.Devices = append(group.Devices, row.Device)
		group.Owner.DeviceCount++
		group.Owner.LiveAddressCount += row.Device.LiveAddressCount
	}
	return groups
}
