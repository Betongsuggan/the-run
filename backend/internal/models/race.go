package models

import "time"

type Race struct {
	ID             string
	EventID        string
	Name           string
	DistanceMeters int
	Discipline     string
	// MaxRunners caps how many runners can self-register via the public
	// form. Zero means unlimited. The admin path bypasses this check.
	MaxRunners int
	// RegistrationFeeOre is the registration fee in öre (1/100 SEK). Zero
	// means the race is free and registrations are implicitly considered
	// paid. Non-zero requires an admin to confirm payment (or, in a future
	// iteration, a Swish webhook) before Registration.PaymentReceivedAt is
	// set.
	RegistrationFeeOre int
	CreatedAt          time.Time
}
