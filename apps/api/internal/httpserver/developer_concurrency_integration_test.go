//go:build integration

package httpserver

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/cryptobox"
)

func TestDeveloperConfigurationWaitsForProtocolTransaction(t *testing.T) {
	f, _, _ := managementFixture(t)
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.applications.Create(t.Context(), user, true, applicationConfig())
	if err != nil {
		t.Fatal(err)
	}
	app := created.Application
	access := f.applicationAccess(t, app.ClientID, "", applicationConfig().RedirectURIs[0])
	box, err := cryptobox.New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	store := authorization.NewStore(f.db.Ent, box)
	ctx, err := store.BeginTX(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Rollback(ctx)
	if _, err := store.GetClient(ctx, app.ClientID); err != nil {
		t.Fatal(err)
	}
	otherContext, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	otherContext, err = store.BeginTX(otherContext)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Rollback(otherContext)
	if _, err := store.GetClient(otherContext, app.ClientID); err != nil {
		t.Fatal("independent protocol reads should remain concurrent")
	}
	if err := store.Rollback(otherContext); err != nil {
		t.Fatal(err)
	}
	blocked, stop := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer stop()
	if _, err := f.applications.SetEnabled(blocked, user, app.ID, app.Revision, false); err == nil || blocked.Err() == nil {
		t.Fatal("configuration changed while a protocol transaction held its snapshot")
	}
	current, err := f.applications.Detail(t.Context(), user.ID, app.ID)
	if err != nil || !current.Enabled || current.Revision != app.Revision {
		t.Fatal("canceled configuration mutation partially committed")
	}
	if err := store.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	app, err = f.applications.SetEnabled(t.Context(), user, app.ID, app.Revision, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.applications.SetEnabled(t.Context(), user, app.ID, app.Revision, true); err != nil {
		t.Fatal(err)
	}
	if f.checkApplicationToken(access) != 401 {
		t.Fatal("reenabling restored a credential from before the configuration boundary")
	}
}
