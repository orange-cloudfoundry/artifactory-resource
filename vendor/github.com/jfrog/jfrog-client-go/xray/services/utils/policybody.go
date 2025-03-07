package utils

import (
	"math"
	"time"
)

type Severity string

const (
	Critical    Severity = "Critical"
	High        Severity = "High"
	Medium      Severity = "Medium"
	Low         Severity = "Low"
	Normal      Severity = "Normal"
	Pending     Severity = "Pending"
	Information Severity = "Information"
	Unknown     Severity = "Unknown"
)

type PolicyType string

const (
	Security PolicyType = "security"
	License  PolicyType = "license"
)

func NewPolicyParams() PolicyParams {
	return PolicyParams{}
}

type PolicyParams struct {
	Name        string
	Type        PolicyType
	Description string
	Rules       []PolicyRule
}

// PolicyBody is the top level payload to be sent to Xray
type PolicyBody struct {
	Name        string       `json:"name,omitempty"`
	Type        PolicyType   `json:"type,omitempty"`
	Description string       `json:"description,omitempty"`
	Author      string       `json:"author,omitempty"`
	Rules       []PolicyRule `json:"rules,omitempty"`
	Created     time.Time    `json:"created,omitempty"`
	Modified    time.Time    `json:"modified,omitempty"`
}

type PolicyRule struct {
	Name     string         `json:"name,omitempty"`
	Criteria PolicyCriteria `json:"criteria,omitempty"`
	Actions  *PolicyAction  `json:"actions,omitempty"`
	Priority int            `json:"priority,omitempty"`
}

type PolicyCriteria struct {
	// Security
	MinSeverity           Severity                `json:"min_severity,omitempty"`
	CvssRange             *PolicyCvssRange        `json:"cvss_range,omitempty"`
	Exposures             *PolicyExposureCriteria `json:"exposures,omitempty"`
	Sast                  *PolicySastCriteria     `json:"sast,omitempty"`
	SkipNotApplicableCVEs bool                    `json:"applicable_cves_only,omitempty"`

	// License
	AllowedLicenses        []string `json:"allowed_licenses,omitempty"`
	BannedLicenses         []string `json:"banned_licenses,omitempty"`
	AllowUnknown           *bool    `json:"allow_unknown,omitempty"`
	MultiLicensePermissive *bool    `json:"multi_license_permissive,omitempty"`
}

type PolicyExposureCriteria struct {
	MinSeverity   Severity `json:"min_severity,omitempty"`
	Secrets       *bool    `json:"secrets,omitempty"`
	Applications  *bool    `json:"applications,omitempty"`
	Services      *bool    `json:"services,omitempty"`
	IaC           *bool    `json:"iac,omitempty"`
	MaliciousCode *bool    `json:"malicious_code,omitempty"`
}

type PolicySastCriteria struct {
	MinSeverity Severity `json:"min_severity,omitempty"`
}

type PolicyCvssRange struct {
	From float64 `json:"from,omitempty"`
	To   float64 `json:"to,omitempty"`
}

type PolicyAction struct {
	Webhooks                       []string            `json:"webhooks,omitempty"`
	BlockDownload                  PolicyBlockDownload `json:"block_download,omitempty"`
	BlockReleaseBundleDistribution *bool               `json:"block_release_bundle_distribution,omitempty"`
	FailBuild                      *bool               `json:"fail_build,omitempty"`
	NotifyDeployer                 *bool               `json:"notify_deployer,omitempty"`
	NotifyWatchRecipients          *bool               `json:"notify_watch_recipients,omitempty"`
	CustomSeverity                 Severity            `json:"custom_severity,omitempty"`
}

type PolicyBlockDownload struct {
	Active    *bool `json:"active,omitempty"`
	Unscanned *bool `json:"unscanned,omitempty"`
}

// Create security policy criteria with min severity
func CreateSeverityPolicyCriteria(minSeverity Severity, skipNotApplicableCves bool) *PolicyCriteria {
	return &PolicyCriteria{
		MinSeverity:           minSeverity,
		SkipNotApplicableCVEs: skipNotApplicableCves,
	}
}

func CreateExposuresPolicyCriteria(minSeverity Severity, secrets, applications, services, iac bool) *PolicyCriteria {
	criteria := &PolicyCriteria{Exposures: &PolicyExposureCriteria{MinSeverity: minSeverity}}
	if secrets {
		criteria.Exposures.Secrets = &secrets
	}
	if applications {
		criteria.Exposures.Applications = &applications
	}
	if services {
		criteria.Exposures.Services = &services
	}
	if iac {
		criteria.Exposures.IaC = &iac
	}
	return criteria
}

func CreateSastPolicyCriteria(minSeverity Severity) *PolicyCriteria {
	return &PolicyCriteria{
		Sast: &PolicySastCriteria{
			MinSeverity: minSeverity,
		},
	}
}

// Create security policy criteria with range.
// from - CVSS range from 0.0 to 10.0
// to - CVSS range from 0.0 to 10.0
func CreateCvssRangePolicyCriteria(from float64, to float64) *PolicyCriteria {
	return &PolicyCriteria{
		CvssRange: &PolicyCvssRange{
			From: math.Round(from*10) / 10,
			To:   math.Round(to*10) / 10,
		},
	}
}

// Create license policy criteria
// allowedLicenses - true if the provided licenses are allowed, false if banned
// allowUnknown - true if should allow unknown licenses, otherwise a violation will be generated for artifacts with unknown licenses
// multiLicensePermissive - do not generate a violation if at least one license is valid in cases whereby multiple licenses were detected on the component
// licenses - the target licenses
func CreateLicensePolicyCriteria(allowedLicenses, allowUnknown, multiLicensePermissive bool, licenses ...string) *PolicyCriteria {
	policyCriteria := &PolicyCriteria{
		AllowUnknown:           &allowUnknown,
		MultiLicensePermissive: &multiLicensePermissive,
	}
	if allowedLicenses {
		policyCriteria.AllowedLicenses = licenses
	} else {
		policyCriteria.BannedLicenses = licenses
	}
	return policyCriteria
}

func CreatePolicyBody(policyParams PolicyParams) PolicyBody {
	return PolicyBody{
		Name:        policyParams.Name,
		Type:        policyParams.Type,
		Description: policyParams.Description,
		Rules:       policyParams.Rules,
	}
}
