package rule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpapi"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

// HTTPHandler handles HTTP requests for rule endpoints.
type HTTPHandler struct {
	ruleService *Service
}

// NewHandler returns a new rule HTTP handler.
func NewHandler(ruleService *Service) *HTTPHandler {
	return &HTTPHandler{
		ruleService: ruleService,
	}
}

// GetDeviceAddressLeaseRule returns the device address lease rule for the device.
func (h *HTTPHandler) GetDeviceAddressLeaseRule(ctx context.Context, request httpapi.GetDeviceAddressLeaseRuleRequestObject) (httpapi.GetDeviceAddressLeaseRuleResponseObject, error) {
	deviceID := device.DeviceID(request.DeviceId)
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "GetDeviceAddressLeaseRule"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
	)

	addressLeaseRule, err := h.ruleService.GetDeviceAddressLeaseRule(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRuleNotFound):
			logger.Warn("rule not found")
			return httpapi.GetDeviceAddressLeaseRule404JSONResponse(errorMsgResponse("Rule not found")), nil
		case errors.Is(err, ErrInvalidRuleConfig):
			logger.Error("invalid rule config detected in db", slog.Any(AttrKeyError, err))
			return httpapi.GetDeviceAddressLeaseRule500JSONResponse(errorMsgResponse("Rule config parsing error")), nil
		default:
			logger.Error("failed to get rule", slog.Any(AttrKeyError, err))
			return httpapi.GetDeviceAddressLeaseRule500JSONResponse(errorMsgResponse("Failed to get rule")), nil
		}
	}

	return httpapi.GetDeviceAddressLeaseRule200JSONResponse(addressLeaseRule.toResponse()), nil
}

// PutDeviceAddressLeaseRule creates or updates the device address lease rule for the device.
func (h *HTTPHandler) PutDeviceAddressLeaseRule(ctx context.Context, request httpapi.PutDeviceAddressLeaseRuleRequestObject) (httpapi.PutDeviceAddressLeaseRuleResponseObject, error) {
	deviceID := device.DeviceID(request.DeviceId)
	ttlSeconds := request.Body.TtlSeconds
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "PutDeviceAddressLeaseRule"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.Int(AttrDeviceAutoExpiryRuleTTL, ttlSeconds),
	)

	r, err := h.ruleService.EnableDeviceAddressLeaseRule(ctx, deviceID, ttlSeconds)
	if err != nil {
		switch {
		case errors.Is(err, device.ErrDeviceNotFound):
			logger.Warn("device not found")
			return httpapi.PutDeviceAddressLeaseRule404JSONResponse(errorMsgResponse(fmt.Sprintf("Device %d not found", deviceID))), nil
		case errors.Is(err, ErrInvalidTTL):
			logger.Warn("invalid ttl value")
			return httpapi.PutDeviceAddressLeaseRule400JSONResponse(errorMsgResponse("ttl_seconds must be at least 1")), nil
		default:
			logger.Error("failed to upsert rule", slog.Any(AttrKeyError, err))
			return httpapi.PutDeviceAddressLeaseRule500JSONResponse(errorMsgResponse("Failed to update rule")), nil
		}
	}
	return httpapi.PutDeviceAddressLeaseRule200JSONResponse(r.toResponse()), nil
}

// DisableDeviceAddressLeaseRule disables the device address lease rule for the device.
func (h *HTTPHandler) DisableDeviceAddressLeaseRule(ctx context.Context, request httpapi.DisableDeviceAddressLeaseRuleRequestObject) (httpapi.DisableDeviceAddressLeaseRuleResponseObject, error) {
	deviceID := device.DeviceID(request.DeviceId)
	ctx, logger := logging.Enrich(ctx,
		slog.String(AttrKeyOperation, "DisableDeviceAddressLeaseRule"),
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
	)

	_, err := h.ruleService.DisableDeviceAddressLeaseRule(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRuleNotFound):
			logger.Info("rule already disabled or missing")
			return httpapi.DisableDeviceAddressLeaseRule204Response{}, nil
		default:
			logger.Error("failed to disable rule", slog.Any(AttrKeyError, err))
			return httpapi.DisableDeviceAddressLeaseRule500JSONResponse(errorMsgResponse("Failed to disable rule")), nil
		}
	}

	return httpapi.DisableDeviceAddressLeaseRule204Response{}, nil
}

func (r *DeviceAddressLeaseRule) toResponse() httpapi.DeviceAddressLeaseRule {
	return httpapi.DeviceAddressLeaseRule{
		Id:         httpapi.ID(r.ID),
		DeviceId:   httpapi.ID(r.DeviceID),
		Enabled:    r.Enabled,
		TtlSeconds: r.Config.TTLSeconds,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
