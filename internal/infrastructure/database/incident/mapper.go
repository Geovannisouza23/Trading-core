package incident

import (
	"trading-core/internal/domain/operation"
	"trading-core/internal/domain/shared"
)

func toDomain(m model) (*operation.Incident, error) {
	id, err := shared.ParseIncidentID(m.ID)
	if err != nil {
		return nil, err
	}
	return &operation.Incident{
		ID:              id,
		Type:            operation.IncidentType(m.Type),
		Severity:        operation.IncidentSeverity(m.Severity),
		Description:     m.Description,
		Source:          m.Source,
		RelatedEntityID: m.RelatedEntityID,
		Status:          operation.IncidentStatus(m.Status),
		DetectedAt:      m.DetectedAt,
		ResolvedAt:      m.ResolvedAt,
		Resolution:      m.Resolution,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}, nil
}

func toModel(i *operation.Incident) model {
	return model{
		ID:              i.ID.String(),
		Type:            string(i.Type),
		Severity:        string(i.Severity),
		Description:     i.Description,
		Source:          i.Source,
		RelatedEntityID: i.RelatedEntityID,
		Status:          string(i.Status),
		DetectedAt:      i.DetectedAt,
		ResolvedAt:      i.ResolvedAt,
		Resolution:      i.Resolution,
		CreatedAt:       i.CreatedAt,
		UpdatedAt:       i.UpdatedAt,
	}
}
