package auth

import (
	"github.com/pquerna/otp/totp"
)

// TOTPIssuer is the label that appears in authenticator apps. Keep it stable
// so re-enrolling the same email doesn't duplicate entries.
const TOTPIssuer = "Ingmarsöloppet"

// GenerateTOTP creates a new shared secret for the given account email and
// returns both the secret (base32) and the otpauth:// URI for QR rendering.
func GenerateTOTP(email string) (secret, otpauthURI string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      TOTPIssuer,
		AccountName: email,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// VerifyTOTP returns true if the 6-digit code matches the secret within the
// current window (±1 step by default in the pquerna library, ~30s).
func VerifyTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}
