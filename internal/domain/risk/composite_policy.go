package risk

// DefaultSpecifications returns the full mandatory specification set in a
// deterministic evaluation order (cheapest/most-decisive checks first).
func DefaultSpecifications() []Specification {
	return []Specification{
		BrokerHealthySpecification{},
		MarketDataFreshSpecification{},
		SignalNotExpiredSpecification{},
		ValidStopSpecification{},
		NoCriticalEventSpecification{},
		MaxDrawdownSpecification{},
		MaxDailyLossSpecification{},
		MaxWeeklyLossSpecification{},
		MaxOpenPositionsSpecification{},
		MaxLeverageSpecification{},
		SufficientBalanceSpecification{},
		MaximumSlippageSpecification{},
	}
}

// CompositeRiskPolicy combines independent specifications into a single
// allow/block decision, collecting every violated reason code rather than
// short-circuiting on the first failure so operators see the full picture.
type CompositeRiskPolicy struct {
	specifications []Specification
}

// NewCompositeRiskPolicy builds a policy from an explicit specification set.
func NewCompositeRiskPolicy(specifications []Specification) *CompositeRiskPolicy {
	return &CompositeRiskPolicy{specifications: specifications}
}

// NewDefaultRiskPolicy builds a policy using DefaultSpecifications.
func NewDefaultRiskPolicy() *CompositeRiskPolicy {
	return NewCompositeRiskPolicy(DefaultSpecifications())
}

// Evaluation is the outcome of running every specification in the policy.
type Evaluation struct {
	Allowed      bool
	ReasonCodes  []ReasonCode
	AppliedRules []string
}

// Evaluate runs every specification against ctx and aggregates the result.
func (p *CompositeRiskPolicy) Evaluate(ctx Context) Evaluation {
	eval := Evaluation{Allowed: true}
	for _, spec := range p.specifications {
		eval.AppliedRules = append(eval.AppliedRules, spec.Name())
		result := spec.Evaluate(ctx)
		if !result.Satisfied {
			eval.Allowed = false
			eval.ReasonCodes = append(eval.ReasonCodes, result.ReasonCode)
		}
	}
	return eval
}
