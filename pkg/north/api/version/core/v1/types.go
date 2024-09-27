package v1

type VersionInfo struct {
	ReleaseVersion string            `json:"releaseVersion,omitempty"`
	GitVersion     string            `json:"gitVersion,omitempty"`
	Fun            map[string]string `json:"fun,omitempty"`
}
