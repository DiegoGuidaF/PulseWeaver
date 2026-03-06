package rule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

// HTTPHandler handles HTTP requests for rule endpoints.
type HTTPHandler struct {
	ruleService *Service
	logger      *slog.Logger
}

// NewHTTPHandler returns a new rule HTTP handler.
func NewHTTPHandler(ruleService *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		ruleService: ruleService,
		logger:      logger.With(slog.String(logging.AttrKeyComponent, "rule")),
	}
}

// GetDeviceAddressLeaseRule returns the device address lease rule for the device.
func (h *HTTPHandler) GetDeviceAddressLeaseRule(ctx context.Context, request httpapi.GetDeviceAddressLeaseRuleRequestObject) (httpapi.GetDeviceAddressLeaseRuleResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDeviceAddressLeaseRule")
	deviceID := device.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	addressLeaseRule, err := h.ruleService.GetDeviceAddressLeaseRule(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRuleNotFound):
			logger.WarnContext(ctx, "rule not found")
			return httpapi.GetDeviceAddressLeaseRule404JSONResponse(errorMsgResponse("Rule not found")), nil
		case errors.Is(err, ErrInvalidRuleConfig):
			logger.ErrorContext(ctx, "invalid rule config detected in db", slog.Any(AttrKeyError, err))
			return httpapi.GetDeviceAddressLeaseRule500JSONResponse(errorMsgResponse("Rule config parsing error")), nil
		default:
			logger.ErrorContext(ctx, "failed to get rule", slog.Any(AttrKeyError, err))
			return httpapi.GetDeviceAddressLeaseRule500JSONResponse(errorMsgResponse("Failed to get rule")), nil
		}
	}

	return httpapi.GetDeviceAddressLeaseRule200JSONResponse(addressLeaseRule.toResponse()), nil
}

// PutDeviceAddressLeaseRule creates or updates the device address lease rule for the device.
func (h *HTTPHandler) PutDeviceAddressLeaseRule(ctx context.Context, request httpapi.PutDeviceAddressLeaseRuleRequestObject) (httpapi.PutDeviceAddressLeaseRuleResponseObject, error) {
	ctx = logging.WithOperation(ctx, "PutDeviceAddressLeaseRule")
	deviceID := device.DeviceID(request.DeviceId)
	ttlSeconds := request.Body.TtlSeconds
	logger := h.logger.With(
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.Int(AttrDeviceAutoExpiryRuleTTL, ttlSeconds),
	)

	r, err := h.ruleService.EnableDeviceAddressLeaseRule(ctx, deviceID, ttlSeconds)
	if err != nil {
		switch {
		case errors.Is(err, device.ErrDeviceNotFound):
			logger.WarnContext(ctx, "device not found")
			return httpapi.PutDeviceAddressLeaseRule404JSONResponse(errorMsgResponse(fmt.Sprintf("Device %d not found", deviceID))), nil
		case errors.Is(err, ErrInvalidTTL):
			logger.WarnContext(ctx, "invalid ttl value")
			return httpapi.PutDeviceAddressLeaseRule400JSONResponse(errorMsgResponse("ttl_seconds must be at least 1")), nil
		default:
			logger.ErrorContext(ctx, "failed to upsert rule", slog.Any(AttrKeyError, err))
			return httpapi.PutDeviceAddressLeaseRule500JSONResponse(errorMsgResponse("Failed to update rule")), nil
		}
	}

	logger.InfoContext(ctx, "device address lease rule updated", slog.Int64(AttrKeyRuleID, int64(r.ID)))

	return httpapi.PutDeviceAddressLeaseRule200JSONResponse(r.toResponse()), nil
}

// DisableDeviceAddressLeaseRule disables the device address lease rule for the device.
func (h *HTTPHandler) DisableDeviceAddressLeaseRule(ctx context.Context, request httpapi.DisableDeviceAddressLeaseRuleRequestObject) (httpapi.DisableDeviceAddressLeaseRuleResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DisableDeviceAddressLeaseRule")
	deviceID := device.DeviceID(request.DeviceId)
	logger := h.logger.With(slog.Int64(AttrKeyDeviceID, deviceID.Int64()))

	rule, err := h.ruleService.DisableDeviceAddressLeaseRule(ctx, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRuleNotFound):
			logger.InfoContext(ctx, "rule already disabled or missing")
			return httpapi.DisableDeviceAddressLeaseRule204Response{}, nil
		default:
			logger.ErrorContext(ctx, "failed to disable rule", slog.Any(AttrKeyError, err))
			return httpapi.DisableDeviceAddressLeaseRule500JSONResponse(errorMsgResponse("Failed to disable rule")), nil
		}
	}
	logger.InfoContext(ctx, "device address lease rule disabled", slog.Int64(AttrKeyRuleID, int64(rule.ID)))

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
