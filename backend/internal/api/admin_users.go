package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/BirgerRydback/the-run/backend/internal/audit"
	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/email"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

// adminInvitationTokenTTL is how long an invitation link is redeemable for.
// 7 days mirrors the choice made in the original plan: long enough to survive
// a weekend, short enough that a stale link doesn't sit around for months.
const adminInvitationTokenTTL = 7 * 24 * time.Hour

// adminInvitePasswordMinLen matches the create-admin CLI: anything shorter
// would let an inviter set a weaker password than we accept via the CLI.
const adminInvitePasswordMinLen = 12

// ── DTOs ──────────────────────────────────────────────────────────────────

// AdminAccountDTO is the projection of an admin user shown on /admin/users.
// PasswordHash / MFASecret are deliberately not exposed.
type AdminAccountDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Locale      string `json:"locale,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	LastLoginAt string `json:"lastLoginAt,omitempty"`
}

// AdminInvitationDTO is the projection of a pending invite. Reads `pending`,
// `expired`, or `used` depending on the row state. `tokenId` is the
// invitation's primary key (the SHA-256 hash); used for revoke calls.
type AdminInvitationDTO struct {
	TokenID       string `json:"tokenId"`
	Email         string `json:"email"`
	Locale        string `json:"locale,omitempty"`
	InvitedByMail string `json:"invitedByMail,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
	ExpiresAt     string `json:"expiresAt"`
	UsedAt        string `json:"usedAt,omitempty"`
}

func adminAccountToDTO(a models.Account) AdminAccountDTO {
	dto := AdminAccountDTO{
		ID:        a.ID,
		Email:     a.Email,
		Locale:    a.Locale,
		CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
	}
	if a.LastLoginAt != nil {
		dto.LastLoginAt = a.LastLoginAt.UTC().Format(time.RFC3339)
	}
	return dto
}

func adminInvitationToDTO(inv models.AdminInvitation, now time.Time) AdminInvitationDTO {
	status := "pending"
	switch {
	case inv.UsedAt != nil:
		status = "used"
	case now.After(inv.ExpiresAt):
		status = "expired"
	}
	dto := AdminInvitationDTO{
		TokenID:       inv.TokenHash,
		Email:         inv.Email,
		Locale:        inv.Locale,
		InvitedByMail: inv.InvitedByMail,
		Status:        status,
		CreatedAt:     inv.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:     inv.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if inv.UsedAt != nil {
		dto.UsedAt = inv.UsedAt.UTC().Format(time.RFC3339)
	}
	return dto
}

type adminAccountListOutput struct {
	Body struct {
		Admins []AdminAccountDTO `json:"admins"`
	}
}

type adminInvitationListOutput struct {
	Body struct {
		Invitations []AdminInvitationDTO `json:"invitations"`
	}
}

type createInvitationInput struct {
	Body struct {
		Email  string `json:"email" format:"email" maxLength:"254"`
		Locale string `json:"locale,omitempty" enum:"sv,en" doc:"Language for the invite email and the new admin's account default. Defaults to 'sv'."`
	}
}

type createInvitationOutput struct {
	Body AdminInvitationDTO
}

type revokeInvitationInput struct {
	TokenID string `path:"tokenId"`
}

type acceptInvitationPreviewInput struct {
	Token string `query:"token" minLength:"1"`
}

type acceptInvitationPreviewOutput struct {
	Body struct {
		Email         string `json:"email"`
		Locale        string `json:"locale,omitempty"`
		InvitedByMail string `json:"invitedByMail,omitempty"`
	}
}

type acceptInvitationInput struct {
	Body struct {
		Token    string `json:"token" minLength:"1"`
		Password string `json:"password" minLength:"12" maxLength:"256"`
	}
}

type acceptInvitationOutput struct {
	Body struct {
		OK    bool   `json:"ok"`
		Email string `json:"email"`
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────

func registerAdminUsers(api huma.API, s store.Store, authCfg auth.Config, renderer *email.Renderer, recorder audit.Recorder) {
	adminMW := adminMiddlewares(authCfg)

	auditAction := func(ctx context.Context, action, targetID, summary string) {
		actor := "admin"
		sub := auth.Subject(ctx)
		if sub != "" {
			actor = "admin:" + sub
		}
		recorder.Record(ctx, models.AuditRow{
			AccountID:  sub,
			Action:     action,
			Actor:      actor,
			TargetType: "admin-account",
			TargetID:   targetID,
			Summary:    summary,
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "admin-list-admins",
		Method:      "GET",
		Path:        "/admin/users",
		Summary:     "List all admin accounts",
		Description: "Admin-only. PasswordHash and MFASecret are never exposed.",
		Tags:        []string{"admin-users"},
		Middlewares: adminMW,
	}, func(ctx context.Context, _ *struct{}) (*adminAccountListOutput, error) {
		accounts, err := s.ListAccounts(ctx)
		if err != nil {
			return nil, fmt.Errorf("list accounts: %w", err)
		}
		admins := make([]AdminAccountDTO, 0)
		for _, a := range accounts {
			if !a.IsAdmin || a.DeletionPendingUntil != nil {
				continue
			}
			admins = append(admins, adminAccountToDTO(a))
		}
		sort.Slice(admins, func(i, j int) bool {
			return admins[i].CreatedAt < admins[j].CreatedAt
		})
		out := &adminAccountListOutput{}
		out.Body.Admins = admins
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin-list-invitations",
		Method:      "GET",
		Path:        "/admin/users/invitations",
		Summary:     "List pending + recently-used admin invitations",
		Tags:        []string{"admin-users"},
		Middlewares: adminMW,
	}, func(ctx context.Context, _ *struct{}) (*adminInvitationListOutput, error) {
		invs, err := s.ListAdminInvitations(ctx)
		if err != nil {
			return nil, fmt.Errorf("list admin invitations: %w", err)
		}
		now := time.Now().UTC()
		out := &adminInvitationListOutput{}
		out.Body.Invitations = make([]AdminInvitationDTO, len(invs))
		for i, inv := range invs {
			out.Body.Invitations[i] = adminInvitationToDTO(inv, now)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "admin-create-invitation",
		Method:        "POST",
		Path:          "/admin/users/invitations",
		Summary:       "Invite a new admin by email",
		Description:   "Generates a 7-day magic link and emails it to the recipient. The recipient sets their own password on /admin/accept and enrolls TOTP on first login.",
		Tags:          []string{"admin-users"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *createInvitationInput) (*createInvitationOutput, error) {
		addr, err := mail.ParseAddress(in.Body.Email)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("email is not valid")
		}
		normalized := models.NormalizeEmail(addr.Address)
		locale := strings.TrimSpace(in.Body.Locale)
		if locale == "" {
			locale = email.LocaleSv
		}

		// Reject inviting an email that already belongs to an active admin —
		// avoids confusion where the recipient already has access.
		existing, err := s.GetAccountByEmail(ctx, normalized)
		if err != nil {
			return nil, fmt.Errorf("lookup existing account: %w", err)
		}
		if existing != nil && existing.IsAdmin && existing.DeletionPendingUntil == nil {
			return nil, huma.Error409Conflict("an admin account already exists for that email")
		}

		// Resolve the inviting admin's email so the template can show it.
		inviterID := auth.Subject(ctx)
		var inviterMail string
		if inviterID != "" {
			if inviter, err := s.GetAccountByID(ctx, inviterID); err == nil && inviter != nil {
				inviterMail = inviter.Email
			}
		}

		rawToken, err := newInvitationTokenID()
		if err != nil {
			return nil, fmt.Errorf("generate invitation token: %w", err)
		}
		tokenHash := hashInvitationToken(rawToken)
		now := time.Now().UTC()
		expires := now.Add(adminInvitationTokenTTL)

		inv := models.AdminInvitation{
			TokenHash:     tokenHash,
			Email:         normalized,
			Locale:        locale,
			InvitedByID:   inviterID,
			InvitedByMail: inviterMail,
			ExpiresAt:     expires,
			CreatedAt:     now,
		}
		if err := s.CreateAdminInvitation(ctx, inv); err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				return nil, fmt.Errorf("invitation token collision (retry): %w", err)
			}
			return nil, fmt.Errorf("store admin invitation: %w", err)
		}

		base := os.Getenv("SITE_BASE_URL")
		if base == "" {
			base = "https://running.rydback.net"
		}
		link := base + "/admin/accept?token=" + rawToken

		if err := renderer.Send(ctx, email.SendOptions{
			Slug:   models.EmailTemplateSlugAdminInvite,
			To:     normalized,
			Locale: locale,
			Vars: struct {
				Link          string
				InvitedByMail string
			}{Link: link, InvitedByMail: inviterMail},
		}); err != nil {
			// IDs only — never the raw token in CloudWatch.
			log.Printf("admin invite email failed: tokenHash=%s err=%v", tokenHash, err)
		}

		auditAction(ctx, "admin.invite.created", normalized, "Invited new admin")

		dto := adminInvitationToDTO(inv, now)
		return &createInvitationOutput{Body: dto}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "admin-revoke-invitation",
		Method:        "DELETE",
		Path:          "/admin/users/invitations/{tokenId}",
		Summary:       "Revoke a pending admin invitation",
		Tags:          []string{"admin-users"},
		DefaultStatus: http.StatusNoContent,
		Middlewares:   adminMW,
	}, func(ctx context.Context, in *revokeInvitationInput) (*struct{}, error) {
		if err := s.DeleteAdminInvitation(ctx, in.TokenID); err != nil {
			return nil, fmt.Errorf("delete admin invitation: %w", err)
		}
		auditAction(ctx, "admin.invite.revoked", in.TokenID, "Revoked admin invitation")
		return nil, nil
	})

	// ── Public accept endpoints (no admin middleware) ────────────────────

	huma.Register(api, huma.Operation{
		OperationID:   "admin-accept-preview",
		Method:        "GET",
		Path:          "/admin/accept/preview",
		Summary:       "Read the email + inviter for a pending admin invitation",
		Description:   "Public. Lets the /admin/accept page show the invited address before the recipient sets their password.",
		Tags:          []string{"admin-users"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *acceptInvitationPreviewInput) (*acceptInvitationPreviewOutput, error) {
		inv, err := lookupValidInvitation(ctx, s, in.Token, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		out := &acceptInvitationPreviewOutput{}
		out.Body.Email = inv.Email
		out.Body.Locale = inv.Locale
		out.Body.InvitedByMail = inv.InvitedByMail
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "admin-accept",
		Method:        "POST",
		Path:          "/admin/accept",
		Summary:       "Accept an admin invitation",
		Description:   "Creates an admin account from a pending invitation. The new admin then signs in normally and enrolls TOTP on first login.",
		Tags:          []string{"admin-users"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, in *acceptInvitationInput) (*acceptInvitationOutput, error) {
		if len(in.Body.Password) < adminInvitePasswordMinLen {
			return nil, huma.Error422UnprocessableEntity(fmt.Sprintf("password must be at least %d characters", adminInvitePasswordMinLen))
		}
		now := time.Now().UTC()
		inv, err := lookupValidInvitation(ctx, s, in.Body.Token, now)
		if err != nil {
			return nil, err
		}

		// Re-check email isn't taken — guards against a runner registering
		// the same address between invite-create and invite-accept.
		existing, err := s.GetAccountByEmail(ctx, inv.Email)
		if err != nil {
			return nil, fmt.Errorf("lookup existing: %w", err)
		}
		if existing != nil {
			return nil, huma.Error409Conflict("an account already exists for that email")
		}

		hash, err := auth.HashPassword(in.Body.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		account := models.Account{
			ID:           uuid.NewString(),
			Email:        inv.Email,
			PasswordHash: hash,
			IsAdmin:      true,
			Locale:       inv.Locale,
			CreatedAt:    now,
		}
		if err := s.CreateAccount(ctx, account); err != nil {
			if errors.Is(err, store.ErrAlreadyExists) {
				return nil, huma.Error409Conflict("an account already exists for that email")
			}
			return nil, fmt.Errorf("create account: %w", err)
		}

		// Mark the invitation used; failure here is non-fatal — the
		// invitation TTL purges it eventually and a second click would
		// just collide on the email sentinel.
		if err := s.MarkAdminInvitationUsed(ctx, inv.TokenHash, now); err != nil {
			log.Printf("admin accept: mark invitation used failed: err=%v", err)
		}

		auditAction(ctx, "admin.invite.accepted", account.ID, "Accepted admin invitation")

		out := &acceptInvitationOutput{}
		out.Body.OK = true
		out.Body.Email = account.Email
		return out, nil
	})
}

// lookupValidInvitation returns the invitation iff it exists, is unused, and
// hasn't expired. Surfaces precise 404/410 status so the accept UI can show
// the right message.
func lookupValidInvitation(ctx context.Context, s store.Store, rawToken string, now time.Time) (*models.AdminInvitation, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, huma.Error404NotFound("invitation token missing")
	}
	hash := hashInvitationToken(rawToken)
	inv, err := s.GetAdminInvitationByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error404NotFound("invitation not found")
		}
		return nil, fmt.Errorf("lookup invitation: %w", err)
	}
	if inv == nil {
		return nil, huma.Error404NotFound("invitation not found")
	}
	if inv.UsedAt != nil {
		return nil, huma.Error410Gone("invitation already used")
	}
	if now.After(inv.ExpiresAt) {
		return nil, huma.Error410Gone("invitation expired")
	}
	return inv, nil
}

// newInvitationTokenID returns 32 bytes (256 bits) of randomness base64url-
// encoded. The renderer puts the raw value in the email link; the store keeps
// only the SHA-256 hash.
func newInvitationTokenID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashInvitationToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
