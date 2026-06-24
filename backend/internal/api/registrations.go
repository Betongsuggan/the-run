package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type RegisterInput struct {
	Body struct {
		Name        string `json:"name" minLength:"1" maxLength:"120" doc:"Full name of the registrant"`
		DateOfBirth string `json:"dateOfBirth" format:"date" doc:"Birth date (YYYY-MM-DD)"`
		Gender      string `json:"gender" enum:"M,F,X" doc:"Gender code"`
		RaceID      string `json:"raceId" minLength:"1" doc:"ID of the race to register for"`
		// Honeypot field: legitimate clients leave it empty. Real bot
		// protection (Turnstile / hCaptcha) is a TODO — see PROJECT_PLAN.md.
		Website string `json:"website,omitempty" doc:"Leave blank — honeypot"`
	}
}

type RegisterOutput struct {
	Body struct {
		ID     string `json:"id" doc:"Server-assigned registration ID"`
		Status string `json:"status" doc:"Lifecycle status, e.g. 'received'"`
	}
}

func registerRegistrations(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "register-for-race",
		Method:        "POST",
		Path:          "/registrations",
		Summary:       "Register for a race",
		Description:   "Public race registration. No authentication required; the race must belong to an event in the future.",
		Tags:          []string{"registrations"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *RegisterInput) (*RegisterOutput, error) {
		if strings.TrimSpace(in.Body.Website) != "" {
			// Honeypot tripped — pretend success so the bot doesn't probe further.
			out := &RegisterOutput{}
			out.Body.ID = "ignored"
			out.Body.Status = "received"
			return out, nil
		}

		dob, err := time.Parse("2006-01-02", in.Body.DateOfBirth)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("dateOfBirth must be YYYY-MM-DD")
		}
		if dob.After(time.Now()) {
			return nil, huma.Error422UnprocessableEntity("dateOfBirth cannot be in the future")
		}

		// TODO(B1): once race + event tables exist, validate that the race
		// exists and its event date is today or later, then persist the
		// registration. For now we log and return a stub ID so the frontend
		// flow can be exercised end-to-end.
		log.Printf("registration received: name=%q dob=%s gender=%s raceId=%s",
			in.Body.Name, in.Body.DateOfBirth, in.Body.Gender, in.Body.RaceID)

		out := &RegisterOutput{}
		out.Body.ID = fmt.Sprintf("reg-%d", time.Now().UnixNano())
		out.Body.Status = "received"
		return out, nil
	})
}
