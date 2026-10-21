package federation

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (p *Provider) github(ctx context.Context, token string) (Profile, error) {
	var user struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
	}
	if err := p.get(ctx, "/user", token, &user); err != nil || user.ID <= 0 {
		return Profile{}, ErrProvider
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := p.get(ctx, "/user/emails", token, &emails); err != nil {
		return Profile{}, err
	}
	profile := Profile{Provider: "github", Issuer: "https://github.com", Subject: strconv.FormatInt(user.ID, 10), DisplayName: displayName(user.Name, user.Login)}
	for _, email := range emails {
		if email.Verified && (profile.Email == "" || email.Primary) {
			profile.Email = strings.TrimSpace(email.Email)
			if email.Primary {
				break
			}
		}
	}
	return profile, nil
}

func displayName(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && utf8.ValidString(value) && utf8.RuneCountInString(value) <= 80 && !strings.ContainsAny(value, "\r\n\x00") {
			return value
		}
	}
	return ""
}
