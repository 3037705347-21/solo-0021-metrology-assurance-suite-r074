package domain

import (
	"fmt"
	"strings"
	"time"
)

type SealStatus string

const (
	SealActive  SealStatus = "active"
	SealRevoked SealStatus = "revoked"
)

type EvidenceSeal struct {
	ID               string     `json:"id"`
	CaseID           string     `json:"case_id"`
	DeviceID         string     `json:"device_id"`
	MeasurementID    string     `json:"measurement_id"`
	Reviewer         string     `json:"reviewer"`
	Summary          string     `json:"summary"`
	Status           SealStatus `json:"status"`
	SupersedesSealID string     `json:"supersedes_seal_id,omitempty"`
	SealedAt         time.Time  `json:"sealed_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
}

func NewEvidenceSeal(id, caseID, deviceID, measurementID, reviewer, summary, supersedesSealID string, at time.Time) (EvidenceSeal, error) {
	item := EvidenceSeal{
		ID:               strings.TrimSpace(id),
		CaseID:           strings.TrimSpace(caseID),
		DeviceID:         strings.TrimSpace(deviceID),
		MeasurementID:    strings.TrimSpace(measurementID),
		Reviewer:         strings.TrimSpace(reviewer),
		Summary:          strings.TrimSpace(summary),
		Status:           SealActive,
		SupersedesSealID: strings.TrimSpace(supersedesSealID),
		SealedAt:         at.UTC(),
	}
	if err := item.Validate(); err != nil {
		return EvidenceSeal{}, err
	}
	return item, nil
}

func (s EvidenceSeal) Validate() error {
	if s.ID == "" || s.CaseID == "" || s.DeviceID == "" || s.MeasurementID == "" || s.Reviewer == "" || s.Summary == "" {
		return fmt.Errorf("%w: seal id, case, device, measurement, reviewer, and summary are required", ErrInvalidInput)
	}
	if s.SealedAt.IsZero() {
		return fmt.Errorf("%w: seal time is required", ErrInvalidInput)
	}
	if s.SupersedesSealID != "" && s.SupersedesSealID == s.ID {
		return fmt.Errorf("%w: a seal cannot supersede itself", ErrInvalidInput)
	}
	switch s.Status {
	case SealActive:
		if s.RevokedAt != nil {
			return fmt.Errorf("%w: an active seal cannot carry a revocation time", ErrInvalidInput)
		}
	case SealRevoked:
		if s.RevokedAt == nil {
			return fmt.Errorf("%w: a revoked seal must carry its revocation time", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unsupported seal status %q", ErrInvalidInput, s.Status)
	}
	return nil
}

func (s *EvidenceSeal) Revoke(at time.Time) error {
	if s.Status != SealActive {
		return fmt.Errorf("%w: only an active seal can be revoked", ErrStateConflict)
	}
	stamp := at.UTC()
	s.Status = SealRevoked
	s.RevokedAt = &stamp
	return nil
}

type SealRevocation struct {
	ID                     string    `json:"id"`
	SealID                 string    `json:"seal_id"`
	CaseID                 string    `json:"case_id"`
	Actor                  string    `json:"actor"`
	Reason                 string    `json:"reason"`
	SupersededMeasurements int       `json:"superseded_measurements"`
	RevokedAt              time.Time `json:"revoked_at"`
}

func NewSealRevocation(id, sealID, caseID, actor, reason string, supersededMeasurements int, at time.Time) (SealRevocation, error) {
	item := SealRevocation{
		ID:                     strings.TrimSpace(id),
		SealID:                 strings.TrimSpace(sealID),
		CaseID:                 strings.TrimSpace(caseID),
		Actor:                  strings.TrimSpace(actor),
		Reason:                 strings.TrimSpace(reason),
		SupersededMeasurements: supersededMeasurements,
		RevokedAt:              at.UTC(),
	}
	if err := item.Validate(); err != nil {
		return SealRevocation{}, err
	}
	return item, nil
}

func (r SealRevocation) Validate() error {
	if r.ID == "" || r.SealID == "" || r.CaseID == "" || r.Actor == "" || r.Reason == "" {
		return fmt.Errorf("%w: revocation id, seal, case, actor, and reason are required", ErrInvalidInput)
	}
	if r.SupersededMeasurements < 0 {
		return fmt.Errorf("%w: superseded measurement count cannot be negative", ErrInvalidInput)
	}
	if r.RevokedAt.IsZero() {
		return fmt.Errorf("%w: revocation time is required", ErrInvalidInput)
	}
	return nil
}
