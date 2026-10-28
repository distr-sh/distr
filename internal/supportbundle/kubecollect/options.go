package kubecollect

type Options struct {
	LogTailLines int64 `json:"logTailLines"`
	// MaxResourceBytes caps every file of the bundle, matching what Distr accepts per resource.
	MaxResourceBytes     int  `json:"maxResourceBytes"`
	IncludeConfigMapData bool `json:"includeConfigMapData"`
	// IncludeSecretData uploads Secret values decoded. Service account and bootstrap tokens are never
	// included, and Helm release Secrets are left out entirely.
	IncludeSecretData bool `json:"includeSecretData"`
}

func DefaultOptions() Options {
	return Options{
		LogTailLines:         1000,
		MaxResourceBytes:     1024 * 1024,
		IncludeConfigMapData: true,
	}
}

// Groups are the parts of a bundle the customer confirms one by one.
type Groups struct {
	Cluster      bool `json:"cluster"`
	Resources    bool `json:"resources"`
	HelmReleases bool `json:"helmReleases"`
	// HelmValues is only set together with HelmReleases.
	HelmValues bool `json:"helmValues"`
	Events     bool `json:"events"`
	Logs       bool `json:"logs"`
}

func (g Groups) Any() bool {
	return g.Cluster || g.Resources || g.HelmReleases || g.Events || g.Logs
}
