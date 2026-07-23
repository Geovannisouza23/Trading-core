package operation

import domainoperation "trading-core/internal/domain/operation"

func toDomain(m model) *domainoperation.State {
	return domainoperation.RehydrateState(domainoperation.Mode(m.CurrentMode), m.ChangedBy, m.Origin, m.Reason, m.ChangedAt, m.Version)
}

func toModel(s *domainoperation.State) model {
	return model{
		CurrentMode: string(s.CurrentMode),
		ChangedBy:   s.ChangedBy,
		Origin:      s.Origin,
		Reason:      s.Reason,
		ChangedAt:   s.ChangedAt,
		Version:     s.Version,
	}
}
