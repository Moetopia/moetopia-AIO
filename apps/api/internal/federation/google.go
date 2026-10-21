package federation

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

func (p *Provider) google(ctx context.Context, token *oauth2.Token, nonce string) (Profile, error) {
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return Profile{}, ErrProvider
	}
	id, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return Profile{}, ErrProvider
	}
	var claims struct {
		AuthorizedParty string `json:"azp"`
		Name            string `json:"name"`
		Email           string `json:"email"`
		EmailVerified   bool   `json:"email_verified"`
	}
	if id.Claims(&claims) != nil || id.Subject == "" || len(id.Subject) > 255 || id.Nonce != nonce || len(id.Audience) != 1 || claims.AuthorizedParty != "" && claims.AuthorizedParty != p.oauth.ClientID || id.IssuedAt.IsZero() || id.IssuedAt.After(time.Now().Add(time.Minute)) || id.IssuedAt.Before(time.Now().Add(-10*time.Minute)) || id.AccessTokenHash != "" && id.VerifyAccessToken(token.AccessToken) != nil {
		return Profile{}, ErrProvider
	}
	profile := Profile{Provider: "google", Issuer: "https://accounts.google.com", Subject: id.Subject, DisplayName: displayName(claims.Name)}
	if claims.EmailVerified {
		profile.Email = claims.Email
	}
	return profile, nil
}
