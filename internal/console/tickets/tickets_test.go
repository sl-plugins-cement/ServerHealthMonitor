package tickets

import "testing"

func TestStoreScopesTicketsAndUpdates(t *testing.T) {
	store := NewStore(t.TempDir())
	created, err := store.Create("服务异常", "无法访问服务接口", "high", "alice")
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if len(store.List("bob", false)) != 0 || len(store.List("alice", false)) != 1 || len(store.List("bob", true)) != 1 {
		t.Fatal("ticket visibility scope is incorrect")
	}
	updated, err := store.Update(created.ID, StatusInProgress, "operator")
	if err != nil || updated.Status != StatusInProgress || updated.Assignee != "operator" {
		t.Fatalf("update ticket: %+v, %v", updated, err)
	}
}
