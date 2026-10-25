package api

type DashboardArtifact struct {
	Artifact            ArtifactResponse `json:"artifact"`
	LatestPulledVersion string           `json:"latestPulledVersion"`
}

type ArtifactsByCustomer struct {
	Customer  CustomerOrganization `json:"customer"`
	Artifacts []DashboardArtifact  `json:"artifacts,omitempty"`
}
