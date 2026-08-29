package auth

import "testing"

func TestRoleAssignmentBoundaries(t *testing.T) {
	if CanAssignRole(RoleAdmin, RoleOwner) || CanAssignRole(RoleAdmin, RoleDirector) || CanAssignRole(RoleAdmin, RoleCustom) {
		t.Fatal("admin must not grant owner, director, or custom roles")
	}
	if !CanAssignRole(RoleOwner, RoleOwner) || !CanAssignRole(RoleOwner, RoleDirector) || !CanAssignRole(RoleDirector, RoleCustom) {
		t.Fatal("owner/director role assignment policy is too restrictive")
	}
}

func TestCustomPermissionsCannotGrantCoreAdministration(t *testing.T) {
	a := NewAuthManager(t.TempDir())
	if _, err := a.CreateUser("custom-user", "Strong-pass-123", RoleCustom); err != nil {
		t.Fatalf("create custom user: %v", err)
	}
	if err := a.SetCustomPermissions("custom-user", []Permission{PermissionViewServers, PermissionManageUsers}); err == nil {
		t.Fatal("custom group must not receive core user-management permission")
	}
}
