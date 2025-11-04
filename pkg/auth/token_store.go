package auth

// Note: Token storage is now handled internally by clambin/tado/v2 library.
// The NewOAuth2Client function automatically handles:
// - Loading encrypted tokens from disk
// - Saving encrypted tokens with passphrase
// - Token refresh and renewal
//
// This file is kept for any future utility functions related to tokens.
