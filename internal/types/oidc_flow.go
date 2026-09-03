package types

type OIDCFlow string

const (
	OIDCFlowLogin        OIDCFlow = "login"
	OIDCFlowRegistration OIDCFlow = "registration"
)

func ParseOIDCFlow(value string) OIDCFlow {
	if value == string(OIDCFlowRegistration) {
		return OIDCFlowRegistration
	}
	return OIDCFlowLogin
}
