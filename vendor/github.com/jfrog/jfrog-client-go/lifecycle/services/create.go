package services

const (
	releaseBundleBaseApi = "api/v2/release_bundle"
)

type SourceType string

const (
	Aql            SourceType = "aql"
	Artifacts      SourceType = "artifacts"
	Builds         SourceType = "builds"
	ReleaseBundles SourceType = "release_bundles"
	Packages       SourceType = "packages"
)

type createOperation struct {
	reqBody        RbCreationBody
	params         CommonOptionalQueryParams
	signingKeyName string
}

func (c *createOperation) getOperationRestApi() string {
	return releaseBundleBaseApi
}

func (c *createOperation) getRequestBody() any {
	return c.reqBody
}

func (c *createOperation) getOperationSuccessfulMsg() string {
	return "Release Bundle successfully created"
}

func (c *createOperation) getOperationParams() CommonOptionalQueryParams {
	return c.params
}

func (c *createOperation) getSigningKeyName() string {
	return c.signingKeyName
}

func (rbs *ReleaseBundlesService) CreateFromAql(rbDetails ReleaseBundleDetails, params CommonOptionalQueryParams, signingKeyName string, aqlQuery string) error {
	return rbs.CreateReleaseBundle(rbDetails, params, signingKeyName, Aql, CreateFromAqlSource{Aql: aqlQuery})
}

func (rbs *ReleaseBundlesService) CreateFromArtifacts(rbDetails ReleaseBundleDetails, params CommonOptionalQueryParams, signingKeyName string, sourceArtifacts CreateFromArtifacts) error {
	return rbs.CreateReleaseBundle(rbDetails, params, signingKeyName, Artifacts, sourceArtifacts)
}

func (rbs *ReleaseBundlesService) CreateFromBuilds(rbDetails ReleaseBundleDetails, params CommonOptionalQueryParams, signingKeyName string, sourceBuilds CreateFromBuildsSource) error {
	return rbs.CreateReleaseBundle(rbDetails, params, signingKeyName, Builds, sourceBuilds)
}

func (rbs *ReleaseBundlesService) CreateFromBundles(rbDetails ReleaseBundleDetails, params CommonOptionalQueryParams, signingKeyName string, sourceReleaseBundles CreateFromReleaseBundlesSource) error {
	return rbs.CreateReleaseBundle(rbDetails, params, signingKeyName, ReleaseBundles, sourceReleaseBundles)
}

func (rbs *ReleaseBundlesService) CreateFromPackages(rbDetails ReleaseBundleDetails, params CommonOptionalQueryParams, signingKeyName string, sourcePackages CreateFromPackagesSource) error {
	return rbs.CreateReleaseBundle(rbDetails, params, signingKeyName, Packages, sourcePackages)
}

func (rbs *ReleaseBundlesService) CreateReleaseBundleFromMultipleSources(rbDetails ReleaseBundleDetails, params CommonOptionalQueryParams,
	signingKeyName string, sources []RbSource) (response []byte, err error) {
	operation := createOperation{
		reqBody: RbCreationBody{
			ReleaseBundleDetails: rbDetails,
			Sources:              sources,
		},
		params:         params,
		signingKeyName: signingKeyName,
	}
	response, err = rbs.doPostOperation(&operation)
	return response, err
}

func (rbs *ReleaseBundlesService) CreateReleaseBundle(rbDetails ReleaseBundleDetails, params CommonOptionalQueryParams,
	signingKeyName string, rbSourceType SourceType, source interface{}) error {
	operation := createOperation{
		reqBody: RbCreationBody{
			ReleaseBundleDetails: rbDetails,
			SourceType:           rbSourceType,
			Source:               source},
		params:         params,
		signingKeyName: signingKeyName,
	}
	_, err := rbs.doPostOperation(&operation)
	return err
}

type CreateFromAqlSource struct {
	Aql string `json:"aql,omitempty"`
}

type SourceBuildDetails struct {
	BuildName   string
	BuildNumber string
	ProjectKey  string
}

type CreateFromArtifacts struct {
	Artifacts []ArtifactSource `json:"artifacts,omitempty"`
}

type CreateFromBuildsSource struct {
	Builds []BuildSource `json:"builds,omitempty"`
}

type CreateFromPackagesSource struct {
	Packages []PackageSource `json:"packages,omitempty"`
}

type ArtifactSource struct {
	Path   string `json:"path,omitempty"`
	Sha256 string `json:"sha256,omitempty"`
}

type BuildSource struct {
	BuildName           string `json:"build_name,omitempty"`
	BuildNumber         string `json:"build_number,omitempty"`
	BuildRepository     string `json:"build_repository,omitempty"`
	IncludeDependencies bool   `json:"include_dependencies,omitempty"`
}

type PackageSource struct {
	PackageName    string `json:"package_name,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	PackageType    string `json:"package_type,omitempty"`
	RepositoryKey  string `json:"repository_key,omitempty"`
}

type CreateFromReleaseBundlesSource struct {
	ReleaseBundles []ReleaseBundleSource `json:"release_bundles,omitempty"`
}

type ReleaseBundleSource struct {
	ReleaseBundleName    string `json:"release_bundle_name,omitempty"`
	ReleaseBundleVersion string `json:"release_bundle_version,omitempty"`
	ProjectKey           string `json:"project,omitempty"`
}

type RbSource struct {
	SourceType     SourceType            `json:"source_type"`
	Builds         []BuildSource         `json:"builds,omitempty"`
	ReleaseBundles []ReleaseBundleSource `json:"release_bundles,omitempty"`
	Artifacts      []ArtifactSource      `json:"artifacts,omitempty"`
	Packages       []PackageSource       `json:"packages,omitempty"`
	Aql            string                `json:"aql,omitempty"`
}
type RbCreationBody struct {
	ReleaseBundleDetails
	SourceType SourceType  `json:"source_type,omitempty"`
	Source     interface{} `json:"source,omitempty"`
	Sources    []RbSource  `json:"sources,omitempty"`
}
