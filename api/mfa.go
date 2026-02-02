package api

type SetupMFAResponse struct {
	Secret    string `json:"secret"`
	QRCodeUrl string `json:"qrCodeUrl"`
}

type EnableMFARequest struct {
	Code string `json:"code"`
}

type EnableMFAResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

type VerifyMFARequest struct {
	Code string `json:"code"`
}

type DisableMFARequest struct {
	Password string `json:"password"`
}

type RegenerateMFARecoveryCodesRequest struct {
	Password string `json:"password"`
}

type RegenerateMFARecoveryCodesResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

type MFARecoveryCodesStatusResponse struct {
	RemainingCodes int `json:"remainingCodes"`
}
