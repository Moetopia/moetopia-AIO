//go:build integration

package httpserver

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/developer"
	"moetopia.local/aio/api/internal/persistence/ent/protocolevent"
)

func TestApplicationStatisticsRealProtocolAndAtomicFailure(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	config := applicationConfig()
	config.Scopes = append(config.Scopes, "offline_access")
	created := decodeAccount[developer.Created](t, f.request("POST", "/api/v1/developer/applications", cookie, csrf, portalOrigin, map[string]any{"public": false, "config": config}), 200)
	app, secret := created.Application, created.ClientSecret
	path := "/api/v1/developer/applications/" + app.ID + "/statistics"
	stats := decodeAccount[authorization.ProtocolStatistics](t, f.request("GET", path, cookie, "", "", nil), 200)
	if len(stats.Days) != 30 || stats.Timezone != "UTC" || stats.RecordingSince.IsZero() || stats.GeneratedAt.IsZero() || stats.Days[0].Counts != nil || stats.Days[29].Counts == nil || !stats.Days[29].Partial || stats.Totals != (authorization.ProtocolCounts{}) {
		t.Fatal("statistics invented unrecorded history or omitted initial coverage")
	}
	if f.request("GET", path, f.identityCookie, "", "", nil).Code != 401 {
		t.Fatal("statistics accepted central identity cookie")
	}
	_, other, _ := managementFixture(t)
	if f.request("GET", path, other, "", "", nil).Code != 404 {
		t.Fatal("statistics exposed foreign app")
	}
	code, verifier := f.testMemberCode(t, app.ClientID, config.RedirectURIs[0])
	if _, err := f.db.SQL.ExecContext(t.Context(), `ALTER TABLE "authorization".protocol_events ADD CONSTRAINT test_statistics_failure CHECK (kind <> 'code_exchange') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	drop := func() {
		if _, err := f.db.SQL.ExecContext(context.Background(), `ALTER TABLE "authorization".protocol_events DROP CONSTRAINT IF EXISTS test_statistics_failure`); err != nil {
			t.Error("could not remove statistics failure fixture")
		}
	}
	defer drop()
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier}, "redirect_uri": {config.RedirectURIs[0]}}
	if f.memberProtocol("/oauth/token", app.ClientID, secret, form).Code != 503 {
		t.Fatal("failed observation did not roll back issuance")
	}
	stats = decodeAccount[authorization.ProtocolStatistics](t, f.request("GET", path, cookie, "", "", nil), 200)
	if stats.Totals.Authorizations != 1 || stats.Totals.CodeExchanges != 0 {
		t.Fatal("failed issuance counted as success")
	}
	drop()
	tokens := decodeAccount[struct {
		Access  string `json:"access_token"`
		Refresh string `json:"refresh_token"`
	}](t, f.memberProtocol("/oauth/token", app.ClientID, secret, form), 200)
	if tokens.Access == "" || tokens.Refresh == "" {
		t.Fatal("failed issuance consumed authorization code")
	}
	decodeAccount[struct {
		Access string `json:"access_token"`
	}](t, f.memberProtocol("/oauth/token", app.ClientID, secret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.Refresh}}), 200)
	stats = decodeAccount[authorization.ProtocolStatistics](t, f.request("GET", path, cookie, "", "", nil), 200)
	if stats.Totals != (authorization.ProtocolCounts{Authorizations: 1, CodeExchanges: 1, Refreshes: 1}) || *stats.Days[29].Counts != stats.Totals {
		t.Fatal("protocol statistics did not count committed grant types exactly")
	}
	if err := f.protocol.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	if count, err := f.db.Ent.ProtocolEvent.Query().Where(protocolevent.ClientID(app.ClientID)).Count(t.Context()); err != nil || count != 3 {
		t.Fatal("protocol cleanup removed current statistics")
	}
}

func TestApplicationStatisticsUTCWindowAndRetention(t *testing.T) {
	f, cookie, csrf := managementFixture(t)
	created := decodeAccount[developer.Created](t, f.request("POST", "/api/v1/developer/applications", cookie, csrf, portalOrigin, map[string]any{"public": true, "config": applicationConfig()}), 200)
	app := created.Application
	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	if _, err := f.db.SQL.ExecContext(t.Context(), `UPDATE "authorization".clients SET statistics_since=$1 WHERE id=$2`, today.AddDate(0, 0, -40), app.ClientID); err != nil {
		t.Fatal(err)
	}
	for _, event := range []struct {
		day  time.Time
		kind protocolevent.Kind
	}{{today.AddDate(0, 0, -30), protocolevent.KindAuthorization}, {today.AddDate(0, 0, -29), protocolevent.KindAuthorization}, {today.AddDate(0, 0, -1), protocolevent.KindCodeExchange}, {today, protocolevent.KindRefresh}, {today, protocolevent.KindMachineToken}} {
		if err := f.db.Ent.ProtocolEvent.Create().SetID(uuid.NewString()).SetClientID(app.ClientID).SetDay(event.day.Format(time.DateOnly)).SetKind(event.kind).Exec(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	path := "/api/v1/developer/applications/" + app.ID + "/statistics"
	stats := decodeAccount[authorization.ProtocolStatistics](t, f.request("GET", path, cookie, "", "", nil), 200)
	if stats.WindowStart != today.AddDate(0, 0, -29).Format(time.DateOnly) || stats.WindowEnd != today.AddDate(0, 0, 1).Format(time.DateOnly) || stats.Totals != (authorization.ProtocolCounts{Authorizations: 1, CodeExchanges: 1, Refreshes: 1, MachineTokens: 1}) || stats.Days[0].Counts.Authorizations != 1 || stats.Days[1].Counts == nil || *stats.Days[1].Counts != (authorization.ProtocolCounts{}) || stats.Days[1].Partial {
		t.Fatal("statistics UTC boundaries, totals or zero filling incorrect")
	}
	if err := f.protocol.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	if count, err := f.db.Ent.ProtocolEvent.Query().Where(protocolevent.ClientID(app.ClientID)).Count(t.Context()); err != nil || count != 4 {
		t.Fatal("retention did not preserve exact current UTC window")
	}
	user, err := f.identity.User(t.Context(), f.userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.applications.SetEnabled(t.Context(), user, app.ID, app.Revision, false); err != nil {
		t.Fatal(err)
	}
	if f.request("GET", path, cookie, "", "", nil).Code != 200 {
		t.Fatal("disabled application lost historical statistics")
	}
	if err := f.applications.Delete(t.Context(), user, app.ID, app.Revision+1); err != nil {
		t.Fatal(err)
	}
	if f.request("GET", path, cookie, "", "", nil).Code != 404 {
		t.Fatal("deleted application statistics remained accessible")
	}
}
