package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

func TestHasEmployeeMapping(t *testing.T) {
	tests := []struct {
		name string
		user *models.User
		want bool
	}{
		{
			name: "nil user",
			user: nil,
			want: false,
		},
		{
			name: "null employee id",
			user: &models.User{Employees: sql.NullInt64{Valid: false}},
			want: false,
		},
		{
			name: "zero employee id",
			user: &models.User{Employees: sql.NullInt64{Int64: 0, Valid: true}},
			want: false,
		},
		{
			name: "valid employee id",
			user: &models.User{Employees: sql.NullInt64{Int64: 42, Valid: true}},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasEmployeeMapping(tc.user)
			if got != tc.want {
				t.Fatalf("hasEmployeeMapping() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRoleFromRightsUsesCanonicalRoles(t *testing.T) {
	tests := []struct {
		name   string
		rights string
		want   string
	}{
		{name: "canonical national", rights: "National Admin", want: utilities.RoleNationalAdmin},
		{name: "canonical national lowercase", rights: "national admin", want: utilities.RoleNationalAdmin},
		{name: "canonical facility", rights: "Facility Admin", want: utilities.RoleFacilityAdmin},
		{name: "canonical staff", rights: "Staff", want: utilities.RoleStaff},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := roleFromRights(tc.rights)
			if got != tc.want {
				t.Fatalf("roleFromRights(%q) = %q, want %q", tc.rights, got, tc.want)
			}
		})
	}
}

func TestGetSessionDataUnauthenticatedReturnsNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sessionManager := scs.New()

	var got any
	h := sessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		got = Get_Session_Data(c, nil, sessionManager, nil)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got != nil {
		t.Fatalf("Get_Session_Data() = %#v, want nil for unauthenticated request", got)
	}
}
