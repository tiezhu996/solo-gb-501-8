package constants

type DecisionType string

const (
	DecisionRelease    DecisionType = "release"
	DecisionQuarantine DecisionType = "quarantine"
	DecisionRework     DecisionType = "rework"
)

func (d DecisionType) Valid() bool {
	switch d {
	case DecisionRelease, DecisionQuarantine, DecisionRework:
		return true
	default:
		return false
	}
}

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleInspector Role = "inspector"
	RoleApprover  Role = "approver"
	RoleOperator  Role = "operator"
	RoleViewer    Role = "viewer"
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleInspector, RoleApprover, RoleOperator, RoleViewer:
		return true
	default:
		return false
	}
}

func (r Role) Can(permission string) bool {
	grants := map[Role]map[string]bool{
		RoleAdmin:     {"line:write": true, "batch:write": true, "inspection:write": true, "release:write": true, "audit:read": true},
		RoleInspector: {"inspection:write": true, "audit:read": true},
		RoleApprover:  {"release:write": true, "audit:read": true},
		RoleOperator:  {"line:write": true, "batch:write": true},
		RoleViewer:    {},
	}
	if r == RoleAdmin {
		return true
	}
	return grants[r][permission]
}
