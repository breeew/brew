package v1

import (
	"context"

	"github.com/breeew/brew-api/pkg/security"
)

const (
	TOKEN_CONTEXT_KEY = "__brew.access_token"
	LANGUAGE_KEY      = "__brew.accept_language"
)

// InjectTokenClaim get user/platform token claims from context
func InjectTokenClaim(ctx context.Context) (security.TokenClaims, bool) {
	val, ok := ctx.Value(TOKEN_CONTEXT_KEY).(security.TokenClaims)
	return val, ok
}

const SPACEID_CONTEXT_KEY = "__brew.spaceid"

func InjectSpaceID(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(SPACEID_CONTEXT_KEY).(string)
	return val, ok
}

func InjectLanguage(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(LANGUAGE_KEY).(string)
	return val, ok
}
