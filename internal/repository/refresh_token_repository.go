package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"

	"contactless-fingerprint-backend/internal/model"
)

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Store persists a new refresh token's hash. Called right after
// auth.GenerateRefreshToken() produces the raw token -- the raw token goes
// to the client in the response, only its hash comes here.
func (r *RefreshTokenRepository) Store(operatorID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT INTO refresh_tokens (token_id, operator_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)
	`, uuid.New().String(), operatorID, tokenHash, expiresAt)
	return err
}

// FindValid looks up a refresh token by its hash, but only returns it if
// it's still usable: not expired, and not revoked. This single WHERE clause
// is the entire "is this refresh token still good?" check for the whole
// system -- everything else (JWT signature checks) is for the access token.
func (r *RefreshTokenRepository) FindValid(tokenHash string) (*model.RefreshToken, error) {
	rt := &model.RefreshToken{}
	err := r.db.QueryRow(`
		SELECT token_id, operator_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > ?
	`, tokenHash, time.Now().UTC()).Scan(
		&rt.TokenID, &rt.OperatorID, &rt.TokenHash,
		&rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// Revoke marks a single refresh token as no longer usable. Used both for
// logout, and for rotation -- each time a refresh token is used to get a
// new access token, the OLD refresh token is revoked and a new one issued,
// so a stolen-and-replayed refresh token stops working the moment the
// legitimate client uses theirs first.
func (r *RefreshTokenRepository) Revoke(tokenID string) error {
	_, err := r.db.Exec(`
		UPDATE refresh_tokens SET revoked_at = ?
		WHERE token_id = ? AND revoked_at IS NULL
	`, time.Now().UTC(), tokenID)
	return err
}

// RevokeAllForOperator kills every active session for an operator in one
// shot -- e.g. "this operator's device was lost/stolen, log them out
// everywhere." Not wired to an endpoint yet, but the operator_id index we
// added in the migration makes this cheap when you need it.
func (r *RefreshTokenRepository) RevokeAllForOperator(operatorID string) error {
	_, err := r.db.Exec(`
		UPDATE refresh_tokens SET revoked_at = ?
		WHERE operator_id = ? AND revoked_at IS NULL
	`, time.Now().UTC(), operatorID)
	return err
}
