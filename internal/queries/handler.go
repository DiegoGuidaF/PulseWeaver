package queries

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type HTTPHandler struct {
	repo         *Repository
	policyReader PolicyMapReader
	npProvider   AuditNetworkPoliciesProvider
	logger       *slog.Logger
}

func NewHTTPHandler(repo *Repository, policyReader PolicyMapReader, npProvider AuditNetworkPoliciesProvider, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:         repo,
		policyReader: policyReader,
		npProvider:   npProvider,
		logger:       logger.With(slog.String(logging.AttrKeyComponent, "queries")),
	}
}

func (h *HTTPHandler) ListUsers(
	ctx context.Context,
	_ httpapi.ListUsersRequestObject,
) (httpapi.ListUsersResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListUsers")

	users, err := h.repo.GetAllUsers(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list users", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListUsers500JSONResponse(errorMsgResponse("Failed to list users")), nil
	}

	response := make(httpapi.ListUsers200JSONResponse, 0, len(users))
	for _, u := range users {
		response = append(response, toUserViewResponse(&u))
	}
	return response, nil
}

func toUserViewResponse(u *UserView) httpapi.User {
	return httpapi.User{
		Id:                  u.ID.Int64(),
		Username:            u.Username,
		DisplayName:         u.DisplayName,
		Email:               openapi_types.Email(u.Email),
		Role:                httpapi.UserRole(u.Role),
		MustChangePassword:  new(u.MustChangePassword),
		BypassHostAllowlist: u.BypassHostAllowlist,
		CreatedAt:           httpapi.UTCTime(u.CreatedAt),
	}
}

func (h *HTTPHandler) GetDeviceAddresses(
	ctx context.Context,
	request httpapi.GetDeviceAddressesRequestObject,
) (httpapi.GetDeviceAddressesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDeviceAddresses")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(device.AttrKeyDeviceID, deviceID.Int64()))

	exists, err := h.repo.DeviceExists(ctx, deviceID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check device existence", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}
	if !exists {
		logger.WarnContext(ctx, "device not found")
		return httpapi.GetDeviceAddresses404JSONResponse(
			errorMsgResponse(fmt.Sprintf("Device with id %d not found", deviceID)),
		), nil
	}

	addresses, err := h.repo.GetDeviceAddresses(ctx, deviceID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list device addresses", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDeviceAddresses500JSONResponse(errorMsgResponse("Failed to list device IPs")), nil
	}

	response := make([]httpapi.Address, len(addresses))
	for i := range addresses {
		response[i] = toAddressViewResponse(&addresses[i])
	}
	return httpapi.GetDeviceAddresses200JSONResponse(response), nil
}

func (h *HTTPHandler) GetDevices(
	ctx context.Context,
	_ httpapi.GetDevicesRequestObject,
) (httpapi.GetDevicesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDevices")

	devices, err := h.repo.GetDevices(ctx, nil)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list devices", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDevices500JSONResponse(errorMsgResponse("Failed to list devices")), nil
	}

	response := make([]httpapi.Device, len(devices))
	for i := range devices {
		response[i] = toDeviceViewResponse(&devices[i])
	}
	return httpapi.GetDevices200JSONResponse(response), nil
}

func (h *HTTPHandler) GetDevicesByUser(
	ctx context.Context,
	request httpapi.GetDevicesByUserRequestObject,
) (httpapi.GetDevicesByUserResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDevicesByUser")

	devices, err := h.repo.GetDevicesByUser(ctx, ids.UserID(request.UserId))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			return httpapi.GetDevicesByUser404JSONResponse(errorMsgResponse("User not found")), nil
		default:
			h.logger.ErrorContext(ctx, "failed to list devices by user", slog.Any(logging.AttrKeyError, err))
			return httpapi.GetDevicesByUser500JSONResponse(errorMsgResponse("Failed to list devices")), nil
		}
	}

	response := make([]httpapi.Device, len(devices))
	for i := range devices {
		response[i] = toDeviceViewResponse(&devices[i])
	}
	return httpapi.GetDevicesByUser200JSONResponse(response), nil
}

func (h *HTTPHandler) ListHostGroups(
	ctx context.Context,
	_ httpapi.ListHostGroupsRequestObject,
) (httpapi.ListHostGroupsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListHostGroups")

	groups, err := h.repo.GetHostGroupsDetails(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list host groups failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHostGroups500JSONResponse(errorMsgResponse("Failed to list host groups")), nil
	}

	groupDetails := make([]httpapi.GroupDetailWithUsers, len(groups))
	for i, g := range groups {
		hosts := make([]httpapi.HostSummary, len(g.Hosts))
		for j, host := range g.Hosts {
			hosts[j] = httpapi.HostSummary{Id: host.ID.Int64(), Fqdn: host.FQDN}
		}
		memberIDs := make([]int64, len(g.MemberIDs))
		for j, id := range g.MemberIDs {
			memberIDs[j] = id.Int64()
		}
		groupDetails[i] = httpapi.GroupDetailWithUsers{
			Id:          g.ID.Int64(),
			Name:        g.Name,
			Color:       g.Color,
			Description: g.Description,
			Icon:        g.Icon,
			CreatedAt:   httpapi.UTCTime(g.CreatedAt),
			Hosts:       hosts,
			Users:       nil, //TODO: Add user data
		}
	}

	return httpapi.ListHostGroups200JSONResponse(httpapi.GroupListResponse{Groups: groupDetails}), nil
}

func (h *HTTPHandler) GetAccessLog(
	ctx context.Context,
	request httpapi.GetAccessLogRequestObject,
) (httpapi.GetAccessLogResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLog")

	params := request.Params

	query := NewAccessLogQuery(params)

	rows, total, err := h.repo.ListAccessLog(ctx, query)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list access log", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLog500JSONResponse(errorMsgResponse("Failed to list access log")), nil
	}

	httpRows := make([]httpapi.AccessLogRow, len(rows))
	for i := range rows {
		httpRows[i] = toAccessLogRow(rows[i])
	}

	var nextCursor *int64
	if len(rows) == query.Limit {
		nextCursor = &rows[len(rows)-1].ID
	}

	response := httpapi.AccessLogResponse{
		Total:      total,
		NextCursor: nextCursor,
		Rows:       httpRows,
	}

	return httpapi.GetAccessLog200JSONResponse(response), nil
}

func (h *HTTPHandler) GetAccessLogByCountry(
	ctx context.Context,
	request httpapi.GetAccessLogByCountryRequestObject,
) (httpapi.GetAccessLogByCountryResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetAccessLogByCountry")

	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now
	if request.Params.From != nil {
		from = request.Params.From.UTC()
	}
	if request.Params.To != nil {
		to = request.Params.To.UTC()
	}

	stats, err := h.repo.ListAccessLogStatsByCountry(ctx, from, to)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list access log stats by country", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetAccessLogByCountry500JSONResponse(errorMsgResponse("Failed to list country stats")), nil
	}

	result := make([]httpapi.AccessLogCountryStats, len(stats))
	for i, s := range stats {
		result[i] = httpapi.AccessLogCountryStats{
			CountryCode:   s.CountryCode,
			CountryName:   &s.CountryName,
			ContinentCode: &s.ContinentCode,
			Total:         int(s.Total),
			Allowed:       int(s.Allowed),
			Denied:        int(s.Denied),
		}
	}

	return httpapi.GetAccessLogByCountry200JSONResponse(result), nil
}

func toAccessLogRow(r AccessLogView) httpapi.AccessLogRow {
	var deviceID *int64
	if r.DeviceID != nil {
		deviceID = new(r.DeviceID.Int64())
	}
	var addressID *int64
	if r.AddressID != nil {
		addressID = new(r.AddressID.Int64())
	}

	var asn *int
	if r.ASN != nil {
		asn = new(int(*r.ASN))
	}

	return httpapi.AccessLogRow{
		Id:                r.ID,
		CreatedAt:         httpapi.UTCTime(r.CreatedAt),
		Outcome:           r.Outcome,
		ClientIp:          r.ClientIP,
		DenyReason:        r.DenyReason,
		DeviceId:          deviceID,
		DeviceName:        r.DeviceName,
		AddressId:         addressID,
		XffChain:          r.XFFChain,
		TargetHost:        r.TargetHost,
		TargetUri:         r.TargetURI,
		HttpMethod:        r.HTTPMethod,
		Headers:           r.Headers,
		CountryCode:       r.CountryCode,
		CountryName:       r.CountryName,
		ContinentCode:     r.ContinentCode,
		Asn:               asn,
		AsnOrg:            r.ASNOrg,
		DurationUs:        &r.DurationUs,
		NetworkPolicyId:   r.NetworkPolicyID,
		NetworkPolicyName: r.NetworkPolicyName,
	}
}

func toAddressViewResponse(a *AddressView) httpapi.Address {
	address := httpapi.Address{
		Id:        a.ID.Int64(),
		DeviceId:  a.DeviceID.Int64(),
		Ip:        a.IP,
		IsEnabled: a.IsEnabled,
		CreatedAt: httpapi.UTCTime(a.CreatedAt),
		UpdatedAt: httpapi.UTCTime(a.UpdatedAt),
	}
	if a.ExpiresAt != nil {
		address.ExpiresAt = new(httpapi.UTCTime(*a.ExpiresAt))
	}

	return address
}

func toDeviceViewResponse(d *DeviceView) httpapi.Device {
	var lastSeenAt *httpapi.UTCTime
	if d.LastSeenAt != nil {
		lastSeenAt = new(httpapi.UTCTime(d.LastSeenAt.Time))
	}
	return httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		DeviceType:   httpapi.DeviceDeviceType(d.DeviceType),
		Description:  d.Description,
		Icon:         d.Icon,
		CreatedAt:    httpapi.UTCTime(d.CreatedAt),
		UpdatedAt:    httpapi.UTCTime(d.UpdatedAt),
		ApiKeyPrefix: d.KeyPrefix,
		AddressCount: new(d.AddressCount),
		LastSeenAt:   lastSeenAt,
		OwnerId:      d.OwnerID.Int64(),
		OwnerName:    new(d.OwnerName),
	}
}

func toDeviceListItem(d *DeviceView) httpapi.DeviceListItem {
	return httpapi.DeviceListItem{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		ApiKeyPrefix: d.KeyPrefix,
		Icon:         d.Icon,
		LiveIpCount:  d.AddressCount,
	}
}

func (h *HTTPHandler) GetDevice(ctx context.Context, request httpapi.GetDeviceRequestObject) (httpapi.GetDeviceResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDevice")
	deviceID := ids.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(device.AttrKeyDeviceID, deviceID.Int64()))

	detail, err := h.repo.GetDeviceDetail(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, device.ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.GetDevice404JSONResponse(errorMsgResponse(fmt.Sprintf("Device with id %d not found", deviceID))), nil
		default:
			logger.ErrorContext(ctx, "failed to get device", slog.Any(logging.AttrKeyError, err))
			return httpapi.GetDevice500JSONResponse(errorMsgResponse("Failed to get device")), nil
		}
	}

	return httpapi.GetDevice200JSONResponse(toDeviceDetailResponse(detail)), nil
}

func toDeviceDetailResponse(d *DeviceDetail) httpapi.Device {
	var lastSeenAt *httpapi.UTCTime
	if d.LastSeenAt != nil {
		lastSeenAt = new(httpapi.UTCTime(d.LastSeenAt.Time))
	}
	return httpapi.Device{
		Id:           d.ID.Int64(),
		Name:         d.Name,
		DeviceType:   httpapi.DeviceDeviceType(d.DeviceType),
		Description:  d.Description,
		Icon:         d.Icon,
		CreatedAt:    httpapi.UTCTime(d.CreatedAt),
		UpdatedAt:    httpapi.UTCTime(d.UpdatedAt),
		ApiKeyPrefix: d.KeyPrefix,
		AddressCount: &d.AddressCount,
		LastSeenAt:   lastSeenAt,
		OwnerId:      d.OwnerID.Int64(),
		OwnerName:    new(d.OwnerName),
	}
}

func (h *HTTPHandler) ListHosts(
	ctx context.Context,
	_ httpapi.ListHostsRequestObject,
) (httpapi.ListHostsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListKnownHosts")

	hosts, err := h.repo.GetAllHostsWithGroups(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list known hosts failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHosts500JSONResponse(errorMsgResponse("Failed to list known hosts")), nil
	}

	resp := make([]httpapi.Host, len(hosts))
	for i, host := range hosts {
		groups := make([]httpapi.GroupSummary, len(host.Groups))
		for j, g := range host.Groups {
			groups[j] = httpapi.GroupSummary{Id: g.ID.Int64(), Name: g.Name, Color: g.Color}
		}
		resp[i] = httpapi.Host{
			Id:        host.ID.Int64(),
			Fqdn:      host.FQDN,
			CreatedAt: httpapi.UTCTime(host.CreatedAt),
			Groups:    groups,
		}
	}
	return httpapi.ListHosts200JSONResponse(httpapi.HostListResponse{Hosts: resp}), nil
}

func (h *HTTPHandler) ListHostSuggestions(
	ctx context.Context,
	_ httpapi.ListHostSuggestionsRequestObject,
) (httpapi.ListHostSuggestionsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListHostSuggestions")

	page, err := h.repo.GetHostSuggestionsPage(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list host suggestions failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHostSuggestions500JSONResponse(errorMsgResponse("Failed to list host suggestions")), nil
	}

	suggestions := make([]httpapi.HostSuggestion, len(page.Suggestions))
	for i, s := range page.Suggestions {
		suggestions[i] = httpapi.HostSuggestion{
			Fqdn:        s.FQDN,
			FirstSeen:   httpapi.UTCTime(s.FirstSeen.Time),
			AllowedHits: s.AllowedHits,
			DeniedHits:  s.DeniedHits,
		}
	}

	ignored := make([]httpapi.IgnoredHostSuggestion, len(page.Ignored))
	for i, s := range page.Ignored {
		ignored[i] = httpapi.IgnoredHostSuggestion{
			Id:        s.ID,
			Fqdn:      s.FQDN,
			CreatedAt: httpapi.UTCTime(s.CreatedAt),
		}
	}

	return httpapi.ListHostSuggestions200JSONResponse(httpapi.HostSuggestionsPage{
		Suggestions: suggestions,
		Ignored:     ignored,
	}), nil
}

func (h *HTTPHandler) ListUsersWithAccess(
	ctx context.Context,
	_ httpapi.ListUsersWithAccessRequestObject,
) (httpapi.ListUsersWithAccessResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListUsersWithAccess")

	rows, err := h.repo.ListUserAccessRows(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list users host access failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListUsersWithAccess500JSONResponse(errorMsgResponse("Failed to list users host access")), nil
	}

	resp := make([]httpapi.UserListItem, len(rows))
	for i, u := range rows {
		groups := make([]httpapi.GroupRef, len(u.Groups))
		for j, g := range u.Groups {
			groups[j] = httpapi.GroupRef{Id: g.ID.Int64(), Name: g.Name}
		}
		resp[i] = httpapi.UserListItem{
			Id:              u.ID.Int64(),
			Username:        u.Username,
			DisplayName:     u.DisplayName,
			Role:            httpapi.UserRole(u.Role),
			BypassHostCheck: u.BypassHostCheck,
			DeviceCount:     u.DeviceCount,
			HostCount:       u.HostCount,
			LiveIpCount:     u.LiveIPCount,
			Groups:          groups,
		}
	}
	return httpapi.ListUsersWithAccess200JSONResponse(resp), nil
}

func (h *HTTPHandler) GetUserAccessDetail(
	ctx context.Context,
	request httpapi.GetUserAccessDetailRequestObject,
) (httpapi.GetUserAccessDetailResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetUserAccessDetail")
	userID := ids.UserID(request.UserId)

	accessDetail, err := h.repo.GetUserAccessDetail(ctx, userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return httpapi.GetUserAccessDetail404JSONResponse(errorMsgResponse("User not found")), nil
		}
		h.logger.ErrorContext(ctx, "get user host details failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetUserAccessDetail500JSONResponse(errorMsgResponse("Failed to get user host details")), nil
	}

	groups := make([]httpapi.SubjectGroupDetail, len(accessDetail.GroupOptions))
	for i, g := range accessDetail.GroupOptions {
		// Build hosts
		hosts := make([]httpapi.HostSummary, len(g.Hosts))
		for j, kh := range g.Hosts {
			hosts[j] = httpapi.HostSummary{Id: kh.ID.Int64(), Fqdn: kh.FQDN}
		}

		// Build network policies
		networkPolicies := make([]httpapi.NetworkPolicyRef, len(g.NetworkPolicies))
		for j, np := range g.NetworkPolicies {
			networkPolicies[j] = httpapi.NetworkPolicyRef{Id: np.ID.Int64(), Cidr: np.CIDR, Name: np.Name}
		}

		groups[i] = httpapi.SubjectGroupDetail{
			Color:           g.Color,
			CreatedAt:       httpapi.UTCTime(g.CreatedAt),
			Description:     g.Description,
			Hosts:           hosts,
			Icon:            g.Icon,
			Id:              g.ID.Int64(),
			Name:            g.Name,
			NetworkPolicies: networkPolicies,
			UpdatedAt:       httpapi.UTCTime(g.UpdatedAt),
			Granted:         g.Granted,
		}
	}

	deviceViews, err := h.repo.GetDevicesByUser(ctx, userID)
	if err != nil {
		h.logger.ErrorContext(ctx, "get user devices failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetUserAccessDetail500JSONResponse(errorMsgResponse("Failed to get user host details")), nil
	}
	devices := make([]httpapi.DeviceListItem, len(deviceViews))
	for i := range deviceViews {
		devices[i] = toDeviceListItem(&deviceViews[i])
	}

	return httpapi.GetUserAccessDetail200JSONResponse(httpapi.UserAccessDetail{
		BypassHostCheck: accessDetail.BypassHostCheck,
		Devices:         devices,
		DisplayName:     accessDetail.User.DisplayName,
		Email:           new(openapi_types.Email(accessDetail.User.Email)),
		Groups:          groups,
		Id:              accessDetail.User.ID.Int64(),
		Role:            httpapi.UserRole(accessDetail.User.Role),
		Username:        accessDetail.User.Username,
	}), nil
}

func (h *HTTPHandler) ListNetworkPolicies(
	ctx context.Context,
	_ httpapi.ListNetworkPoliciesRequestObject,
) (httpapi.ListNetworkPoliciesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListNetworkPolicies")

	summaries, err := h.repo.GetNetworkPolicySummaries(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list network policies", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListNetworkPolicies500JSONResponse(errorMsgResponse("Failed to list network policies")), nil
	}

	resp := make(httpapi.ListNetworkPolicies200JSONResponse, len(summaries))
	for i, s := range summaries {
		resp[i] = toNetworkPolicySummaryResponse(s)
	}
	return resp, nil
}

func (h *HTTPHandler) GetNetworkPolicy(
	ctx context.Context,
	request httpapi.GetNetworkPolicyRequestObject,
) (httpapi.GetNetworkPolicyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetNetworkPolicy")

	id := ids.NetworkPolicyID(request.Id)
	detail, err := h.repo.GetNetworkPolicyDetail(ctx, id)
	if err != nil {
		if errors.Is(err, networkpolicies.ErrNotFound) {
			return httpapi.GetNetworkPolicy404JSONResponse(errorMsgResponse("Network policy not found")), nil
		}
		h.logger.ErrorContext(ctx, "failed to get network policy", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetNetworkPolicy500JSONResponse(errorMsgResponse("Failed to get network policy")), nil
	}
	return httpapi.GetNetworkPolicy200JSONResponse(toNetworkPolicyDetailResponse(*detail)), nil
}

func toNetworkPolicySummaryResponse(s NetworkPolicySummaryView) httpapi.NetworkPolicyListItem {
	return httpapi.NetworkPolicyListItem{
		Id:              s.ID.Int64(),
		Name:            s.Name,
		Cidr:            s.CIDR,
		Enabled:         s.Enabled,
		BypassHostCheck: s.AllowAllHosts,
		HostCount:       s.EffectiveHostCount,
		Groups:          []httpapi.GroupRef{},
	}
}

func toNetworkPolicyDetailResponse(d NetworkPolicyDetailView) httpapi.NetworkPolicyDetail {
	groups := make([]httpapi.SubjectGroupDetail, len(d.HostGroups))
	for i, g := range d.HostGroups {
		hosts := make([]httpapi.HostSummary, len(g.Hosts))
		for j, h := range g.Hosts {
			hosts[j] = httpapi.HostSummary{
				Id:   h.ID,
				Fqdn: h.FQDN,
			}
		}
		var color string
		if g.Color != nil {
			color = *g.Color
		}
		var icon string
		if g.Icon != nil {
			icon = *g.Icon
		}
		groups[i] = httpapi.SubjectGroupDetail{
			Id:              g.ID,
			Name:            g.Name,
			Color:           color,
			Icon:            icon,
			Hosts:           hosts,
			Granted:         g.Assigned,
			NetworkPolicies: []httpapi.NetworkPolicyRef{},
		}
	}

	return httpapi.NetworkPolicyDetail{
		Id:              d.ID.Int64(),
		Name:            d.Name,
		Cidr:            d.CIDR,
		Description:     d.Description,
		Enabled:         d.Enabled,
		BypassHostCheck: d.AllowAllHosts,
		Groups:          groups,
		CreatedAt:       httpapi.UTCTime(d.CreatedAt),
		UpdatedAt:       httpapi.UTCTime(d.UpdatedAt),
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
