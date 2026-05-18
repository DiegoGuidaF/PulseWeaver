package ids

import "strconv"

type HostGroupID int64

func (id HostGroupID) Int64() int64   { return int64(id) }
func (id HostGroupID) String() string { return strconv.FormatInt(int64(id), 10) }

type SessionID int64

func (id SessionID) Int64() int64 {
	return int64(id)
}

func (id SessionID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

type UserID int64

func (id UserID) Int64() int64 {
	return int64(id)
}

func (id UserID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

type AddressID int64

func (id AddressID) Int64() int64 {
	return int64(id)
}

func (id AddressID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

type DeviceID int64

func (id DeviceID) Int64() int64 {
	return int64(id)
}

func (id DeviceID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

type KnownHostID int64

func (id KnownHostID) Int64() int64   { return int64(id) }
func (id KnownHostID) String() string { return strconv.FormatInt(int64(id), 10) }

// AddressLeaseID represents the primary key of a row in the address_leases table.
type AddressLeaseID int64

// NetworkPolicyID is a typed alias over int64 for compile-time safety.
type NetworkPolicyID int64

func (id NetworkPolicyID) Int64() int64   { return int64(id) }
func (id NetworkPolicyID) String() string { return strconv.FormatInt(int64(id), 10) }

// RuleID represents the primary key of a row in the device_rules table.
type RuleID int64
