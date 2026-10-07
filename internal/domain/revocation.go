package domain

import (
	"fmt"
	"strings"
	"time"
)

// SealRevocation is the independent fact recording that a reviewer withdrew
// an evidence seal. The original seal is preserved untouched; the revocation
// carries the reason and the responsible reviewer.
type SealRevocation struct {
	ID        string    `json:"id"`
	SealID    string    `json:"seal_id"`
	CaseID    string    `json:"case_id"`
	Reviewer  string    `json:"reviewer"`
	Reason    string    `json:"reason"`
	RevokedAt time.Time `json:"revoked_at"`
}

func NewSealRevocation(id, sealID, caseID, reviewer, reason string, at time.Time) (SealRevocation, error) {
	item := SealRevocation{
		ID:        strings.TrimSpace(id),
		SealID:    strings.TrimSpace(sealID),
		CaseID:    strings.TrimSpace(caseID),
		Reviewer:  strings.TrimSpace(reviewer),
		Reason:    strings.TrimSpace(reason),
		RevokedAt: at.UTC(),
	}
	if err := item.Validate(); err != nil {
		return SealRevocation{}, err
	}
	return item, nil
}

func (r SealRevocation) Validate() error {
	if r.ID == "" || r.SealID == "" || r.CaseID == "" || r.Reviewer == "" || r.Reason == "" {
		return fmt.Errorf("%w: revocation id, seal, case, reviewer, and reason are required", ErrInvalidInput)
	}
	if r.RevokedAt.IsZero() {
		return fmt.Errorf("%w: revocation time is required", ErrInvalidInput)
	}
	return nil
}
