package api

type DashboardArtifact struct {
	Artifact            ArtifactResponse `json:"artifact"`
	LatestPulledVersion string           `json:"latestPulledVersion"`
	IsLatest            bool             `json:"isLatest"`
}

type ArtifactsByCustomer struct {
	Customer  CustomerOrganization `json:"customer"`
	Artifacts []DashboardArtifact  `json:"artifacts,omitempty"`
}
