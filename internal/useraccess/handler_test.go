//go:build test

package useraccess_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_SetUserHostGrants(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	adminID := testutils.AdminPrincipal(t, srv).UserID
	url := fmt.Sprintf("/api/v1/admin/access/users/%d/grants", adminID)

	body, _ := json.Marshal(map[string]any{"bypass_host_check": false, "group_ids": []int{}})
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNoContent)
}
