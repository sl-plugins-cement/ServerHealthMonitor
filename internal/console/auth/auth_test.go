package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short", "alice"); err == nil {
		t.Fatal("expected weak password to be rejected")
	}
	if err := ValidatePassword("Strong-pass-123", "alice"); err != nil {
		t.Fatalf("expected strong password to pass: %v", err)
	}
}

func TestLoginRequiresAPIKeyAndLocksAfterThreeFailures(t *testing.T) {
	a := NewAuthManager(t.TempDir())
	apiKey, err := a.CreateUser("alice", "Strong-pass-123", RoleAdmin)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	withoutKey := a.Login("alice", "Strong-pass-123", "")
	if withoutKey.Success || withoutKey.Remaining != 2 {
		t.Fatalf("expected missing API key to count as a failure: %+v", withoutKey)
	}
	for attempt := 0; attempt < 2; attempt++ {
		result := a.Login("alice", "Strong-pass-123", "wrong-key")
		if result.Success {
			t.Fatal("wrong API key must not authenticate")
		}
	}

	locked := a.Login("alice", "Strong-pass-123", apiKey)
	if locked.Success || !locked.Locked {
		t.Fatalf("expected account to be permanently locked: %+v", locked)
	}
}

func TestLoginReportsMustChangePasswordOnFirstLogin(t *testing.T) {
	a := NewAuthManager(t.TempDir())
	apiKey, err := a.CreateUser("alice", "Strong-pass-123", RoleAdmin)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	first := a.Login("alice", "Strong-pass-123", apiKey)
	if !first.Success || !first.MustChangePassword {
		t.Fatalf("first login must report must-change-password: %+v", first)
	}
	if !a.NeedsPasswordChange("alice") {
		t.Fatal("NeedsPasswordChange must be true right after creation")
	}

	if err := a.ChangeOwnPassword("alice", "Strong-pass-123", "New-pass-456"); err != nil {
		t.Fatalf("change own password: %v", err)
	}
	if a.NeedsPasswordChange("alice") {
		t.Fatal("must-change flag should be cleared after changing the password")
	}

	second := a.Login("alice", "New-pass-456", apiKey)
	if !second.Success || second.MustChangePassword {
		t.Fatalf("login after password change must not require another change: %+v", second)
	}
}

func TestChangeOwnPasswordRequiresCurrentPassword(t *testing.T) {
	a := NewAuthManager(t.TempDir())
	if _, err := a.CreateUser("alice", "Strong-pass-123", RoleAdmin); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := a.ChangeOwnPassword("alice", "wrong-old", "New-pass-456"); err == nil {
		t.Fatal("changing password with a wrong current password must fail")
	}
	if !a.NeedsPasswordChange("alice") {
		t.Fatal("failed change attempt must not clear the must-change flag")
	}
}

func TestAdminPasswordResetForcesChangeNextLogin(t *testing.T) {
	a := NewAuthManager(t.TempDir())
	apiKey, err := a.CreateUser("alice", "Strong-pass-123", RoleAdmin)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := a.ChangeOwnPassword("alice", "Strong-pass-123", "New-pass-456"); err != nil {
		t.Fatalf("change own password: %v", err)
	}
	if a.NeedsPasswordChange("alice") {
		t.Fatal("must-change flag should be cleared after changing the password")
	}

	// Admin reset must force a password change again.
	if err := a.ChangePassword("alice", "Admin-reset-789"); err != nil {
		t.Fatalf("admin reset password: %v", err)
	}
	if !a.NeedsPasswordChange("alice") {
		t.Fatal("admin password reset must require a password change on next login")
	}
	reset := a.Login("alice", "Admin-reset-789", apiKey)
	if !reset.Success || !reset.MustChangePassword {
		t.Fatalf("login after admin reset must require a password change: %+v", reset)
	}
}
