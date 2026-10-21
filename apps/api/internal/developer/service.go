package developer

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"moetopia.local/aio/api/internal/authorization"
	"moetopia.local/aio/api/internal/identity"
	"moetopia.local/aio/api/internal/notifications"
	"moetopia.local/aio/api/internal/persistence"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/applicationversion"
	"moetopia.local/aio/api/internal/persistence/ent/developerapplication"
)

var ErrMissing = errors.New("application unavailable")
var ErrConflict = errors.New("application revision or state changed")
var ErrInvalid = errors.New("invalid application configuration")
var ErrQuota = errors.New("application quota reached")

type Config struct {
	Name         string   `json:"name" minLength:"1" maxLength:"80"`
	Description  string   `json:"description" maxLength:"500"`
	HomepageURL  string   `json:"homepage_url" maxLength:"2048"`
	RedirectURIs []string `json:"redirect_uris" minItems:"1" maxItems:"10"`
	Scopes       []string `json:"scopes" minItems:"1" maxItems:"4"`
}
type Version struct {
	Version      int    `json:"version"`
	Config       Config `json:"config"`
	Status       string `json:"status"`
	ReviewReason string `json:"review_reason"`
}
type Application struct {
	ID               string    `json:"id"`
	ClientID         string    `json:"client_id"`
	Public           bool      `json:"public"`
	Enabled          bool      `json:"enabled"`
	Suspended        bool      `json:"suspended" required:"false"`
	SuspensionReason string    `json:"suspension_reason" required:"false"`
	Revision         int       `json:"revision"`
	ActiveVersion    int       `json:"active_version"`
	Latest           Version   `json:"latest"`
	Approved         *Version  `json:"approved,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
type Created struct {
	Application  Application `json:"application"`
	ClientSecret string      `json:"client_secret,omitempty"`
}
type Service struct {
	db       *ent.Client
	protocol *authorization.Provider
	users    *identity.Service
	issuer   string
	outbox   *notifications.Outbox
}

func New(db *ent.Client, protocol *authorization.Provider, users *identity.Service, issuer string, outbox *notifications.Outbox) *Service {
	return &Service{db, protocol, users, issuer, outbox}
}
func (s *Service) validate(config Config) (Config, error) {
	config.Name, config.Description = strings.TrimSpace(config.Name), strings.TrimSpace(config.Description)
	if !utf8.ValidString(config.Name+config.Description) || utf8.RuneCountInString(config.Name) < 1 || utf8.RuneCountInString(config.Name) > 80 || utf8.RuneCountInString(config.Description) > 500 || len(config.RedirectURIs) < 1 || len(config.RedirectURIs) > 10 || len(config.Scopes) < 1 || len(config.Scopes) > 4 || !slices.Contains(config.Scopes, "openid") {
		return config, ErrInvalid
	}
	seen := map[string]bool{}
	for _, value := range config.RedirectURIs {
		if seen[value] || !s.validURL(value) {
			return config, ErrInvalid
		}
		seen[value] = true
	}
	if config.HomepageURL != "" && !s.validURL(config.HomepageURL) {
		return config, ErrInvalid
	}
	seen = map[string]bool{}
	for _, scope := range config.Scopes {
		if seen[scope] || !slices.Contains([]string{"openid", "profile", "email", "offline_access"}, scope) {
			return config, ErrInvalid
		}
		seen[scope] = true
	}
	return config, nil
}
func (s *Service) validURL(value string) bool {
	u, err := url.Parse(value)
	if err != nil || len(value) > 2048 || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || strings.ContainsAny(value, "\r\n\\") {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	host := u.Hostname()
	return strings.HasPrefix(s.issuer, "http://") && u.Scheme == "http" && (host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost"))
}
func configMap(config Config) map[string]any {
	raw, _ := json.Marshal(config)
	var result map[string]any
	_ = json.Unmarshal(raw, &result)
	return result
}
func publicVersion(row *ent.ApplicationVersion) (Version, error) {
	raw, err := json.Marshal(row.Config)
	if err != nil {
		return Version{}, err
	}
	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return Version{}, err
	}
	return Version{row.Version, config, string(row.Status), row.ReviewReason}, nil
}
func (s *Service) view(ctx context.Context, db *ent.Client, row *ent.DeveloperApplication) (Application, error) {
	rows, err := db.ApplicationVersion.Query().Where(applicationversion.ApplicationID(row.ID), applicationversion.VersionIn(row.LatestVersion, row.ActiveVersion)).All(ctx)
	if err != nil {
		return Application{}, err
	}
	return assemble(row, rows)
}
func assemble(row *ent.DeveloperApplication, versions []*ent.ApplicationVersion) (Application, error) {
	var latest, active *ent.ApplicationVersion
	for _, version := range versions {
		if version.ApplicationID != row.ID {
			continue
		}
		if version.Version == row.LatestVersion {
			latest = version
		}
		if version.Version == row.ActiveVersion {
			active = version
		}
	}
	if latest == nil {
		return Application{}, errors.New("application version unavailable")
	}
	version, err := publicVersion(latest)
	if err != nil {
		return Application{}, err
	}
	result := Application{ID: row.ID, ClientID: row.ClientID, Public: row.Public, Enabled: row.Enabled, Suspended: row.Suspended, SuspensionReason: row.SuspensionReason, Revision: row.Revision, ActiveVersion: row.ActiveVersion, Latest: version, CreatedAt: row.CreatedAt}
	if row.ActiveVersion > 0 {
		if active == nil {
			return result, errors.New("approved version unavailable")
		}
		value, err := publicVersion(active)
		if err != nil {
			return result, err
		}
		result.Approved = &value
	}
	return result, nil
}
func (s *Service) Detail(ctx context.Context, owner, id string) (Application, error) {
	db := persistence.Client(ctx, s.db)
	row, err := db.DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.OwnerID(owner), developerapplication.Deleted(false)).Only(ctx)
	if ent.IsNotFound(err) {
		return Application{}, ErrMissing
	}
	if err != nil {
		return Application{}, err
	}
	return s.view(ctx, db, row)
}
func (s *Service) begin(ctx context.Context, user identity.User, id string, revision int) (context.Context, *persistence.Unit, *ent.DeveloperApplication, error) {
	tx, err := persistence.Begin(ctx, s.db)
	if err != nil {
		return ctx, nil, nil, err
	}
	ctx = persistence.WithClient(ctx, tx.Client())
	if err := s.users.VerifyVersion(ctx, user.ID, user.SecurityVersion); err != nil {
		_ = tx.Rollback()
		return ctx, nil, nil, err
	}
	row, err := tx.DeveloperApplication.Query().Where(developerapplication.ID(id), developerapplication.OwnerID(user.ID), developerapplication.Deleted(false)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		err = ErrMissing
	}
	if err != nil {
		_ = tx.Rollback()
		return ctx, nil, nil, err
	}
	if row.Revision != revision {
		_ = tx.Rollback()
		return ctx, nil, nil, ErrConflict
	}
	return ctx, tx, row, nil
}
func (s *Service) spec(row *ent.DeveloperApplication, config Config, testOnly bool) authorization.ManagedClient {
	return authorization.ManagedClient{ID: row.ClientID, OwnerID: row.OwnerID, Name: config.Name, Public: row.Public, Enabled: row.Enabled && !row.Suspended, TestOnly: testOnly, RedirectURIs: config.RedirectURIs, Scopes: config.Scopes}
}
