# Tado Device Code Flow Implementation

This document describes how tado-prometheus-exporter implements the RFC 8628 Device Code Grant flow for Tado API authentication.

## Reference Documentation

Based on official Tado documentation:
https://support.tado.com/en/articles/8565472-how-do-i-authenticate-to-access-the-rest-api

## Authentication Flow

### Overview

The Device Code Grant flow (RFC 8628) is designed for headless devices and command-line applications where users cannot perform interactive web login on the device itself.

### Step-by-Step Flow

#### 1. Request Device Code

**Endpoint:** `POST https://login.tado.com/oauth2/device_authorize`

**Parameters:**
- `client_id` - Your OAuth2 application ID (provided when registering with Tado)
- `scope` - `offline_access` (to request a refresh token)

**Request:**
```bash
curl -X POST https://login.tado.com/oauth2/device_authorize \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=YOUR_CLIENT_ID&scope=offline_access"
```

**Response:**
```json
{
  "device_code": "ABC123DEF456GHI789",
  "user_code": "WXYZ-1234",
  "verification_uri": "https://my.tado.com/authorize",
  "verification_uri_complete": "https://my.tado.com/authorize?user_code=WXYZ-1234",
  "expires_in": 300,
  "interval": 5
}
```

**Key Timings:**
- Device code expires in 300 seconds (5 minutes)
- Polling interval is 5 seconds

#### 2. Display Authorization Prompt

Display the `user_code` and `verification_uri` to the user:

```
========================================
Authorization Required
========================================
Please visit: https://my.tado.com/authorize
Enter code: WXYZ-1234

Waiting for authorization...
========================================
```

The user visits the URI and logs into their Tado account to authorize access.

#### 3. Poll for Token

**Endpoint:** `POST https://login.tado.com/oauth2/token`

**Parameters:**
- `client_id` - Your OAuth2 application ID
- `client_secret` - Your OAuth2 application secret
- `device_code` - From step 1 response
- `grant_type` - `urn:ietf:params:oauth:grant-type:device_code` (RFC 8628 grant type)

**Request:**
```bash
curl -X POST https://login.tado.com/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=YOUR_CLIENT_ID&client_secret=YOUR_SECRET&device_code=ABC123DEF456GHI789&grant_type=urn:ietf:params:oauth:grant-type:device_code"
```

**Poll Behavior:**
- Poll at intervals specified by `interval` from step 1 (typically 5 seconds)
- Continue until user authorizes or device code expires (5 minutes)
- Each poll returns either:
  - `error: "authorization_pending"` - User hasn't authorized yet
  - `error: other` - Fatal error
  - `access_token` - Success!

**Successful Response:**
```json
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGc...",
  "token_type": "Bearer",
  "expires_in": 600,
  "refresh_token": "eyJ0eXAiOiJKV1QiLCJhbGc..."
}
```

**Key Timings:**
- Access tokens valid for 600 seconds (10 minutes)
- Refresh tokens valid for up to 30 days

#### 4. Use Access Token

Include the access token in API requests:

```bash
curl -H "Authorization: Bearer ACCESS_TOKEN" \
  https://api.tado.com/api/v2/me
```

#### 5. Token Refresh (Automatic)

When access token expires, use the refresh token:

**Endpoint:** `POST https://login.tado.com/oauth2/token`

**Parameters:**
- `client_id`
- `client_secret`
- `refresh_token` - From original token response
- `grant_type` - `refresh_token`

The OAuth2 library handles this automatically via the token source.

## Implementation Details

### Tado-Specific Customizations

Our implementation directly handles the Tado device code flow (RFC 8628) because:

1. **Correct Endpoints**: Uses Tado's actual endpoints (`login.tado.com/oauth2/device_authorize` and `login.tado.com/oauth2/token`)

2. **Custom Grant Type**: Sends the correct RFC 8628 grant type: `urn:ietf:params:oauth:grant-type:device_code`

3. **Direct HTTP Implementation**: Makes raw HTTP calls to ensure compatibility with Tado's specific implementation

4. **Proper Error Handling**: Distinguishes between `authorization_pending` (continue polling) and other errors (fatal)

### Code Structure

**File:** `pkg/auth/authenticator.go`

**Key Functions:**
- `NewDeviceAuthenticator()` - Creates authenticator with Tado endpoints
- `requestDeviceCode()` - Makes HTTP POST to `https://login.tado.com/oauth2/device_authorize`
- `pollForToken()` - Polls for token with proper interval and timeout
- `exchangeDeviceCode()` - Makes HTTP POST to `https://login.tado.com/oauth2/token` with RFC 8628 grant type

**Token Timing:**
- Device code expiry: 5 minutes (enforced by poll timeout)
- Access token validity: 10 minutes
- Refresh token validity: Up to 30 days
- Token expiry buffer: 1 minute (tokens considered expired if within 60 seconds of expiry)

### Configuration

To use with real Tado credentials:

```bash
./exporter \
  --client-id=YOUR_TADO_CLIENT_ID \
  --client-secret=YOUR_TADO_CLIENT_SECRET
```

To get OAuth2 credentials:
1. Register your application with Tado
2. Obtain Client ID and Client Secret
3. Pass them to the exporter

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `invalid_client` | Wrong client ID or secret | Verify credentials with Tado |
| `authorization_pending` | User hasn't authorized yet | Continue polling |
| `invalid_grant` | Device code expired or already used | Request a new device code |
| `timeout waiting for user authorization` | User took longer than 5 minutes | Request a new device code |

### Token-Related Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `token is nil` | No token available after auth | Ensure authorization completed |
| `cannot refresh: no refresh token` | Offline_access scope not requested | Re-authenticate |
| `failed to exchange device code` | Network error during polling | Check connectivity and retry |

## Security Considerations

### Sensitive Data Handling

1. **Client Secret**: Never expose in logs or version control
   - Passed via command-line flag only
   - Never logged or printed

2. **Access Token**: Automatically included in API requests via oauth2 library

3. **Refresh Token**: Stored in persistent token file
   - File permissions: `0600` (read/write for owner only)
   - Default location: `~/.tado-exporter/token.json`
   - Can specify custom location via `--token-path`

### Scope Limitations

The `offline_access` scope requests:
- Read access to home/zone data
- Ability to obtain refresh tokens for long-lived sessions
- No write/delete permissions to user data

## Testing

### Unit Tests

59 tests verify:
- Token store persistence
- Device code flow initiation
- Token reuse from storage
- Expired token handling
- Error cases

Run tests:
```bash
go test -v ./pkg/auth
```

### Integration Tests

Integration tests verify:
- End-to-end token reuse path
- Device code flow initiation (without real Tado auth)
- Authenticated client creation

Run integration tests:
```bash
go test -v ./pkg/auth -run "Integration"
```

### Manual Testing

Test with a saved token:
```bash
# Create test token
mkdir -p ~/.tado-exporter
cat > ~/.tado-exporter/token.json << 'EOF'
{
  "access_token": "test_token",
  "refresh_token": "refresh",
  "token_type": "Bearer",
  "expiry": "2099-01-01T00:00:00Z"
}
EOF

# Run exporter (should reuse token without device code flow)
./exporter --client-id=dummy --client-secret=dummy
```

Test device code flow with real credentials:
```bash
./exporter \
  --client-id=YOUR_REAL_CLIENT_ID \
  --client-secret=YOUR_REAL_CLIENT_SECRET

# Follow the device code authentication prompt
# Token will be saved to ~/.tado-exporter/token.json
```

## References

- **RFC 8628** - OAuth 2.0 Device Authorization Grant: https://datatracker.ietf.org/doc/html/rfc8628
- **Tado API Documentation** - https://support.tado.com/en/articles/8565472-how-do-i-authenticate-to-access-the-rest-api
- **OAuth 2.0** - https://oauth.net/2/

## Future Improvements

1. Support for additional scopes if Tado API expands
2. Custom polling interval configuration
3. Device code flow cancellation
4. Token cache invalidation strategies
