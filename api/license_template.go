package api

type CreateLicenseTemplateRequest struct {
	Name                      string `json:"name"`
	PayloadTemplate           string `json:"payloadTemplate"`
	ExpirationGracePeriodDays int    `json:"expirationGracePeriodDays"`
}

type UpdateLicenseTemplateRequest struct {
	Name                      string `json:"name"`
	PayloadTemplate           string `json:"payloadTemplate"`
	ExpirationGracePeriodDays int    `json:"expirationGracePeriodDays"`
}
