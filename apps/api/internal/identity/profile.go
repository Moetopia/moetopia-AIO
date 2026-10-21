package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/draw"
	"moetopia.local/aio/api/internal/audit"
	"moetopia.local/aio/api/internal/persistence/ent"
	"moetopia.local/aio/api/internal/persistence/ent/accountuser"
)

var ErrConflict = errors.New("account revision changed")
var avatarCapacity = make(chan struct{}, 2)

func (s *Service) UpdateProfile(ctx context.Context, id, name, bio string, revision int) (User, error) {
	name, bio = strings.TrimSpace(name), strings.TrimSpace(bio)
	if revision < 0 || !utf8.ValidString(name) || !utf8.ValidString(bio) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 80 || utf8.RuneCountInString(bio) > 500 {
		return User{}, ErrInvalid
	}
	return s.updateProfile(ctx, id, revision, "profile.updated", func(update *ent.AccountUserUpdate) { update.SetDisplayName(name).SetBio(bio) })
}
func normalizeAvatar(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	if len(encoded) > 700000 {
		return nil, ErrInvalid
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(raw) > 512<<10 {
		return nil, ErrInvalid
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png") || config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || int64(config.Width)*int64(config.Height) > 4_000_000 {
		return nil, ErrInvalid
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrInvalid
	}
	bounds := source.Bounds()
	size := min(bounds.Dx(), bounds.Dy())
	crop := image.Rect(bounds.Min.X+(bounds.Dx()-size)/2, bounds.Min.Y+(bounds.Dy()-size)/2, bounds.Min.X+(bounds.Dx()-size)/2+size, bounds.Min.Y+(bounds.Dy()-size)/2+size)
	avatar := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	draw.CatmullRom.Scale(avatar, avatar.Bounds(), source, crop, draw.Src, nil)
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, avatar); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
func (s *Service) UpdateAvatar(ctx context.Context, id, encoded string, revision int) (User, error) {
	select {
	case avatarCapacity <- struct{}{}:
		defer func() { <-avatarCapacity }()
	case <-ctx.Done():
		return User{}, ctx.Err()
	}
	avatar, err := normalizeAvatar(encoded)
	if err != nil || revision < 0 {
		return User{}, ErrInvalid
	}
	return s.updateProfile(ctx, id, revision, "avatar.updated", func(update *ent.AccountUserUpdate) {
		if len(avatar) == 0 {
			update.ClearAvatar().SetAvatarDigest("")
		} else {
			hash := sha256.Sum256(avatar)
			update.SetAvatar(avatar).SetAvatarDigest(hex.EncodeToString(hash[:]))
		}
	})
}
func (s *Service) updateProfile(ctx context.Context, id string, revision int, action string, apply func(*ent.AccountUserUpdate)) (User, error) {
	tx, err := s.store.db.Tx(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	update := tx.AccountUser.Update().Where(accountuser.ID(id), accountuser.Enabled(true), accountuser.DeletionDueAtIsNil(), accountuser.Revision(revision)).AddRevision(1)
	apply(update)
	count, err := update.Save(ctx)
	if err != nil {
		return User{}, err
	}
	if count != 1 {
		return User{}, ErrConflict
	}
	if err := audit.Record(ctx, tx.Client(), id, action); err != nil {
		return User{}, err
	}
	row, err := tx.AccountUser.Get(ctx, id)
	if err != nil {
		return User{}, err
	}
	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	return public(row), nil
}
func (s *Service) Avatar(ctx context.Context, id string) ([]byte, error) {
	row, err := s.store.db.AccountUser.Query().Where(accountuser.ID(id), accountuser.Enabled(true), accountuser.DeletionDueAtIsNil()).Select(accountuser.FieldAvatar).Only(ctx)
	if ent.IsNotFound(err) || err == nil && len(row.Avatar) == 0 {
		return nil, ErrMissing
	}
	if err != nil {
		return nil, err
	}
	return row.Avatar, nil
}
func (s *Service) SecurityEvents(ctx context.Context, id string) ([]audit.Event, error) {
	return audit.List(ctx, s.store.db, id)
}
