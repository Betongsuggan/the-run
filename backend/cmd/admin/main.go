// Command admin manages admin Accounts directly in DynamoDB without going
// through the auth-protected API. The intended use is bootstrapping the very
// first admin after a fresh deploy:
//
//	just create-admin email=birger@example.com password='…'
//
// or rotating a password:
//
//	go run ./cmd/admin -email birger@example.com -password '…' -force
//
// It honors the same env vars as the API: ACCOUNTS_TABLE_NAME, AWS region,
// AWS_ENDPOINT_URL (LocalStack), etc. Refuses to overwrite an existing account
// unless -force is set.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/mail"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/models"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

func main() {
	emailFlag := flag.String("email", "", "admin email (required)")
	passwordFlag := flag.String("password", "", "admin password (required, min 12 chars)")
	forceFlag := flag.Bool("force", false, "if the account already exists, rotate its password instead of erroring")
	flag.Parse()

	if *emailFlag == "" || *passwordFlag == "" {
		fmt.Fprintln(os.Stderr, "usage: admin -email EMAIL -password PASSWORD [-force]")
		os.Exit(2)
	}
	if len(*passwordFlag) < 12 {
		fmt.Fprintln(os.Stderr, "password must be at least 12 characters")
		os.Exit(2)
	}
	addr, err := mail.ParseAddress(*emailFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid email: %v\n", err)
		os.Exit(2)
	}
	email := models.NormalizeEmail(addr.Address)

	ctx := context.Background()
	s, err := store.NewDynamoStoreFromEnv(ctx)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	hash, err := auth.HashPassword(*passwordFlag)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	existing, err := s.GetAccountByEmail(ctx, email)
	if err != nil {
		log.Fatalf("lookup account: %v", err)
	}

	if existing != nil {
		if !*forceFlag {
			log.Fatalf("account already exists for %s (rerun with -force to rotate)", email)
		}
		existing.PasswordHash = hash
		existing.IsAdmin = true
		// Reset MFA so the next login re-enrolls. Operator should communicate
		// this to the account owner out of band.
		existing.MFASecret = ""
		if err := s.UpdateAccount(ctx, *existing); err != nil {
			log.Fatalf("update account: %v", err)
		}
		fmt.Printf("rotated password for admin account %s (id=%s); MFA reset, will re-enroll on next login\n", email, existing.ID)
		return
	}

	acc := models.Account{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		IsAdmin:      true,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.CreateAccount(ctx, acc); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			log.Fatalf("account already exists (race after lookup); rerun with -force")
		}
		log.Fatalf("create account: %v", err)
	}
	fmt.Printf("created admin account %s (id=%s); enroll TOTP on first login\n", email, acc.ID)
}
