package structs

type DeploymentInfo struct {
	ArtifactURL string `json:"artifact_url"`
	Manifest    string `json:"manifest"`
	Username    string
	Password    string
	Environment string
	Org         string
	Space       string
	AppName     string
	Data        map[string]interface{} `json:"data"`
	UUID        string
	SkipSSL     bool
}
