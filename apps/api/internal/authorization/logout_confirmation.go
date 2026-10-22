package authorization

import (
	"context"
	"errors"
	"net/url"

	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/sessions"
)

func (p *Provider) LogoutDetails(ctx context.Context, handle, browser, subject string) (LogoutDetails, error) {
	row, data, err := p.logoutInteraction(ctx, handle, browser, false)
	if err != nil {
		return LogoutDetails{}, err
	}
	client, err := p.logoutClient(ctx, row.ClientID, data.RedirectURI, false)
	if err != nil {
		return LogoutDetails{}, err
	}
	if subject != "" && data.HintSubject != "" && data.HintSubject != subject {
		return LogoutDetails{}, ErrLogoutAccount
	}
	if err := p.allowLogoutSubject(ctx, client.ID, subject); err != nil {
		return LogoutDetails{}, err
	}
	return LogoutDetails{client.ID, client.Name, data.RedirectURI, row.ExpiresAt, true}, nil
}

func (p *Provider) ConfirmLogout(ctx context.Context, handle, browser, subject, sessionID string, approve bool, web *sessions.Service) (string, error) {
	meta, data, err := p.logoutInteraction(ctx, handle, browser, false)
	if err != nil {
		return "", err
	}
	unit, err := persistence.Begin(ctx, p.store.client)
	if err != nil {
		return "", err
	}
	defer unit.Rollback()
	ctx = persistence.WithClient(ctx, unit.Client())
	if subject != "" && data.HintSubject != "" && data.HintSubject != subject {
		return "", ErrLogoutAccount
	}
	if approve && subject != "" {
		if web == nil || sessionID == "" {
			return "", ErrLogoutAccount
		}
		if err := web.RevokeLive(ctx, subject, sessionID); err != nil {
			return "", err
		}
	}
	client, err := p.logoutClient(ctx, meta.ClientID, data.RedirectURI, true)
	if err != nil {
		return "", err
	}
	row, data, err := p.logoutInteraction(ctx, handle, browser, true)
	if err != nil {
		return "", err
	}
	if subject != "" && data.HintSubject != "" && data.HintSubject != subject {
		return "", ErrLogoutAccount
	}
	if err := p.allowLogoutSubject(ctx, client.ID, subject); err != nil {
		return "", err
	}
	if approve && subject != "" {
		if err := audit.RecordTarget(ctx, unit.Client(), subject, "session.central_logout", sessionID, "success"); err != nil {
			return "", err
		}
	}
	if err := unit.LogoutInteraction.UpdateOneID(row.ID).SetConsumed(true).SetPayload([]byte{}).Exec(ctx); err != nil {
		return "", err
	}
	if err := unit.Commit(); err != nil {
		return "", err
	}
	if !approve {
		return p.issuer + "/auth/logout?cancelled=1", nil
	}
	if data.RedirectURI == "" {
		return p.issuer + "/auth/logout?completed=1", nil
	}
	redirect, err := url.Parse(data.RedirectURI)
	if err != nil {
		return "", ErrLogout
	}
	values := redirect.Query()
	if data.State != "" {
		values.Set("state", data.State)
	}
	redirect.RawQuery = values.Encode()
	return redirect.String(), nil
}

func (p *Provider) allowLogoutSubject(ctx context.Context, clientID, subject string) error {
	if subject == "" {
		return nil
	}
	err := p.AllowSubject(ctx, clientID, subject)
	if errors.Is(err, ErrGrantMissing) {
		return ErrLogoutAccount
	}
	return err
}
