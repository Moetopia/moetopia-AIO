//go:build integration

package authorization

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/administration"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/persistence"
)

func testAdministratorBootstrap(t *testing.T, db *persistence.Database, key string) {
	users, err := identity.New(identity.NewStore(db.Ent, nil), key)
	if err != nil {
		t.Fatal(err)
	}
	if err := administration.Bootstrap(t.Context(), db.Ent, users, "unknown@example.test"); err == nil {
		t.Fatal("bootstrap accepted missing account")
	}
	emails := make([]string, 2)
	for i := range emails {
		id := uuid.NewString()
		emails[i] = id + "@example.test"
		if err := db.Ent.AccountUser.Create().SetID(id).SetEmail(emails[i]).SetDisplayName("Bootstrap fixture").SetPasswordHash("unusable-test-password").SetEmailVerifiedAt(time.Now()).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	var group sync.WaitGroup
	results := make(chan error, 2)
	for _, email := range emails {
		group.Go(func() { results <- administration.Bootstrap(t.Context(), db.Ent, users, email) })
	}
	group.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatal("administrator bootstrap was not single-use under concurrency")
	}
	if count, err := db.Ent.AdminPrincipal.Query().Count(t.Context()); err != nil || count != 1 {
		t.Fatal("bootstrap assigned multiple administrator roles")
	}
	if err := administration.Bootstrap(t.Context(), db.Ent, users, emails[0]); err == nil {
		t.Fatal("bootstrap could be repeated")
	}
}
