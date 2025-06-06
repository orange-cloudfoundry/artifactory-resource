package xray

import (
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	"github.com/jfrog/jfrog-client-go/xray/services"
	xrayUtils "github.com/jfrog/jfrog-client-go/xray/services/utils"
	"github.com/jfrog/jfrog-client-go/xray/services/xsc"
)

// XrayServicesManager defines the http client and general configuration
type XrayServicesManager struct {
	client *jfroghttpclient.JfrogHttpClient
	config config.Config
	// Global reference to the provided project key, used for API endpoints that require it for authentication
	scopeProjectKey string
}

// New creates a service manager to interact with Xray
func New(config config.Config) (*XrayServicesManager, error) {
	details := config.GetServiceDetails()
	var err error
	manager := &XrayServicesManager{config: config}
	manager.client, err = jfroghttpclient.JfrogClientBuilder().
		SetCertificatesPath(config.GetCertificatesPath()).
		SetInsecureTls(config.IsInsecureTls()).
		SetContext(config.GetContext()).
		SetDialTimeout(config.GetDialTimeout()).
		SetOverallRequestTimeout(config.GetOverallRequestTimeout()).
		SetClientCertPath(details.GetClientCertPath()).
		SetClientCertKeyPath(details.GetClientCertKeyPath()).
		AppendPreRequestInterceptor(details.RunPreRequestFunctions).
		SetRetries(config.GetHttpRetries()).
		SetRetryWaitMilliSecs(config.GetHttpRetryWaitMilliSecs()).
		Build()
	return manager, err
}

func (sm *XrayServicesManager) SetProjectKey(projectKey string) *XrayServicesManager {
	sm.scopeProjectKey = projectKey
	return sm
}

// Client will return the http client
func (sm *XrayServicesManager) Client() *jfroghttpclient.JfrogHttpClient {
	return sm.client
}

func (sm *XrayServicesManager) Config() config.Config {
	return sm.config
}

// GetVersion will return the Xray version
func (sm *XrayServicesManager) GetVersion() (string, error) {
	versionService := services.NewVersionService(sm.client)
	versionService.XrayDetails = sm.config.GetServiceDetails()
	return versionService.GetVersion()
}

// CreateWatch will create a new Xray watch
func (sm *XrayServicesManager) CreateWatch(params xrayUtils.WatchParams) error {
	watchService := services.NewWatchService(sm.client)
	watchService.XrayDetails = sm.config.GetServiceDetails()
	return watchService.Create(params)
}

// GetWatch retrieves the details about an Xray watch by name
// It will error if no watch can be found by that name.
func (sm *XrayServicesManager) GetWatch(watchName string) (*xrayUtils.WatchParams, error) {
	watchService := services.NewWatchService(sm.client)
	watchService.XrayDetails = sm.config.GetServiceDetails()
	return watchService.Get(watchName)
}

// UpdateWatch will update an existing Xray watch by name
// It will error if no watch can be found by that name.
func (sm *XrayServicesManager) UpdateWatch(params xrayUtils.WatchParams) error {
	watchService := services.NewWatchService(sm.client)
	watchService.XrayDetails = sm.config.GetServiceDetails()
	return watchService.Update(params)
}

// DeleteWatch will delete an existing watch by name
// It will error if no watch can be found by that name.
func (sm *XrayServicesManager) DeleteWatch(watchName string) error {
	watchService := services.NewWatchService(sm.client)
	watchService.XrayDetails = sm.config.GetServiceDetails()
	return watchService.Delete(watchName)
}

// CreatePolicy will create a new Xray policy
func (sm *XrayServicesManager) CreatePolicy(params xrayUtils.PolicyParams) error {
	policyService := services.NewPolicyService(sm.client)
	policyService.XrayDetails = sm.config.GetServiceDetails()
	return policyService.Create(params)
}

// GetPolicy retrieves the details about an Xray policy by name
// It will error if no policy can be found by that name.
func (sm *XrayServicesManager) GetPolicy(policyName string) (*xrayUtils.PolicyParams, error) {
	policyService := services.NewPolicyService(sm.client)
	policyService.XrayDetails = sm.config.GetServiceDetails()
	return policyService.Get(policyName)
}

// UpdatePolicy will update an existing Xray policy by name
// It will error if no policy can be found by that name.
func (sm *XrayServicesManager) UpdatePolicy(params xrayUtils.PolicyParams) error {
	policyService := services.NewPolicyService(sm.client)
	policyService.XrayDetails = sm.config.GetServiceDetails()
	return policyService.Update(params)
}

// DeletePolicy will delete an existing policy by name
// It will error if no policy can be found by that name.
func (sm *XrayServicesManager) DeletePolicy(policyName string) error {
	policyService := services.NewPolicyService(sm.client)
	policyService.XrayDetails = sm.config.GetServiceDetails()
	return policyService.Delete(policyName)
}

// CreatePolicy will create a new Xray ignore rule
// The function returns the ignore rule id if succeeded or empty string and error message if fails
func (sm *XrayServicesManager) CreateIgnoreRule(params xrayUtils.IgnoreRuleParams) (string, error) {
	ignoreRuleService := services.NewIgnoreRuleService(sm.client)
	ignoreRuleService.XrayDetails = sm.config.GetServiceDetails()
	return ignoreRuleService.Create(params)
}

// CreatePolicy will create a new Xray ignore rule
// The function returns the ignore rule id if succeeded or empty string and error message if fails
func (sm *XrayServicesManager) GetIgnoreRule(ignoreRuleId string) (*xrayUtils.IgnoreRuleParams, error) {
	ignoreRuleService := services.NewIgnoreRuleService(sm.client)
	ignoreRuleService.XrayDetails = sm.config.GetServiceDetails()
	return ignoreRuleService.Get(ignoreRuleId)
}

// CreatePolicy will create a new Xray ignore rule
// The function returns the ignore rule id if succeeded or empty string and error message if fails
func (sm *XrayServicesManager) DeleteIgnoreRule(ignoreRuleId string) error {
	ignoreRuleService := services.NewIgnoreRuleService(sm.client)
	ignoreRuleService.XrayDetails = sm.config.GetServiceDetails()
	return ignoreRuleService.Delete(ignoreRuleId)
}

// AddBuildsToIndexing will add builds to Xray indexing configuration
func (sm *XrayServicesManager) AddBuildsToIndexing(buildNames []string) error {
	binMgrService := services.NewBinMgrService(sm.client)
	binMgrService.XrayDetails = sm.config.GetServiceDetails()
	return binMgrService.AddBuildsToIndexing(buildNames)
}

func (sm *XrayServicesManager) IsTokenValidationEnabled() (isEnabled bool, err error) {
	jasConfigService := services.NewJasConfigService(sm.client)
	jasConfigService.XrayDetails = sm.config.GetServiceDetails()
	return jasConfigService.GetJasConfigTokenValidation()
}

// ScanGraph will send Xray the given graph for scan
// Returns a string represents the scan ID.
func (sm *XrayServicesManager) ScanGraph(params services.XrayGraphScanParams) (scanId string, err error) {
	scanService := services.NewScanService(sm.client)
	scanService.XrayDetails = sm.config.GetServiceDetails()
	scanService.ScopeProjectKey = sm.scopeProjectKey
	return scanService.ScanGraph(params)
}

// GetScanGraphResults returns an Xray scan output of the requested graph scan.
// The scanId input should be received from ScanGraph request.
func (sm *XrayServicesManager) GetScanGraphResults(scanID, xrayVersion string, includeVulnerabilities, includeLicenses, xscEnabled bool) (*services.ScanResponse, error) {
	scanService := services.NewScanService(sm.client)
	scanService.XrayDetails = sm.config.GetServiceDetails()
	scanService.ScopeProjectKey = sm.scopeProjectKey
	return scanService.GetScanGraphResults(scanID, xrayVersion, includeVulnerabilities, includeLicenses, xscEnabled)
}

func (sm *XrayServicesManager) ImportGraph(params services.XrayGraphImportParams, fileName string) (scanId string, err error) {
	enrichService := services.NewEnrichService(sm.client)
	enrichService.XrayDetails = sm.config.GetServiceDetails()
	return enrichService.ImportGraph(params, fileName)
}

// GetScanGraphResults returns an Xray scan output of the requested graph scan.
// The scanId input should be received from ScanGraph request.
func (sm *XrayServicesManager) GetImportGraphResults(scanID string) (*services.ScanResponse, error) {
	enrichService := services.NewEnrichService(sm.client)
	enrichService.XrayDetails = sm.config.GetServiceDetails()
	return enrichService.GetImportGraphResults(scanID)
}

// BuildScan scans a published build-info with Xray.
// 'scanResponse' - Xray scan output of the requested build scan.
// 'noFailBuildPolicy' - Indicates that the Xray API returned a "No Xray Fail build...." error
func (sm *XrayServicesManager) BuildScan(params services.XrayBuildParams, includeVulnerabilities bool) (scanResponse *services.BuildScanResponse, noFailBuildPolicy bool, err error) {
	buildScanService := services.NewBuildScanService(sm.client)
	buildScanService.XrayDetails = sm.config.GetServiceDetails()
	buildScanService.ScopeProjectKey = sm.scopeProjectKey
	return buildScanService.ScanBuild(params, includeVulnerabilities)
}

// GenerateVulnerabilitiesReport returns a Xray report response of the requested report
func (sm *XrayServicesManager) GenerateVulnerabilitiesReport(params services.VulnerabilitiesReportRequestParams) (resp *services.ReportResponse, err error) {
	reportService := services.NewReportService(sm.client)
	reportService.XrayDetails = sm.config.GetServiceDetails()
	return reportService.Vulnerabilities(params)
}

// GenerateLicensesReport returns a Xray report response of the requested report
func (sm *XrayServicesManager) GenerateLicensesReport(params services.LicensesReportRequestParams) (resp *services.ReportResponse, err error) {
	reportService := services.NewReportService(sm.client)
	reportService.XrayDetails = sm.config.GetServiceDetails()
	return reportService.Licenses(params)
}

// GenerateVoilationsReport returns a Xray report response of the requested report
func (sm *XrayServicesManager) GenerateViolationsReport(params services.ViolationsReportRequestParams) (resp *services.ReportResponse, err error) {
	reportService := services.NewReportService(sm.client)
	reportService.XrayDetails = sm.config.GetServiceDetails()
	return reportService.Violations(params)
}

// ReportDetails returns a Xray details response for the requested report
func (sm *XrayServicesManager) ReportDetails(reportId string) (details *services.ReportDetails, err error) {
	reportService := services.NewReportService(sm.client)
	reportService.XrayDetails = sm.config.GetServiceDetails()
	return reportService.Details(reportId)
}

// ReportContent returns a Xray report content response for the requested report
func (sm *XrayServicesManager) ReportContent(params services.ReportContentRequestParams) (content *services.ReportContent, err error) {
	reportService := services.NewReportService(sm.client)
	reportService.XrayDetails = sm.config.GetServiceDetails()
	return reportService.Content(params)
}

// DeleteReport deletes a Xray report
func (sm *XrayServicesManager) DeleteReport(reportId string) error {
	reportService := services.NewReportService(sm.client)
	reportService.XrayDetails = sm.config.GetServiceDetails()
	return reportService.Delete(reportId)
}

// ArtifactSummary returns Xray artifact summaries for the requested checksums and/or paths
func (sm *XrayServicesManager) ArtifactSummary(params services.ArtifactSummaryParams) (*services.ArtifactSummaryResponse, error) {
	summaryService := services.NewSummaryService(sm.client)
	summaryService.XrayDetails = sm.config.GetServiceDetails()
	return summaryService.GetArtifactSummary(params)
}

// IsEntitled returns true if the user is entitled for the requested feature ID
func (sm *XrayServicesManager) IsEntitled(featureId string) (bool, error) {
	entitlementsService := services.NewEntitlementsService(sm.client)
	entitlementsService.XrayDetails = sm.config.GetServiceDetails()
	entitlementsService.ScopeProjectKey = sm.scopeProjectKey
	return entitlementsService.IsEntitled(featureId)
}

// Xsc returns the Xsc service inside Xray
func (sm *XrayServicesManager) Xsc() *xsc.XscInnerService {
	xscService := xsc.NewXscService(sm.client)
	xscService.XrayDetails = sm.config.GetServiceDetails()
	xscService.ScopeProjectKey = sm.scopeProjectKey
	return xscService
}
