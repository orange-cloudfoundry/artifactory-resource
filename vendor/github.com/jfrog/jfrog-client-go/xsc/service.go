package xsc

import "github.com/jfrog/jfrog-client-go/xsc/services"

// XscService defines the API to interact with XSC
type XscService interface {
	// GetVersion will return the Xsc version
	GetVersion() (string, error)
	// AddAnalyticsGeneralEvent will send an analytics metrics general event to Xsc and return MSI (multi scan id) generated by Xsc.
	AddAnalyticsGeneralEvent(event services.XscAnalyticsGeneralEvent, xrayVersion string) (string, error)
	// SendXscLogErrorRequest will send an error log to Xsc
	SendXscLogErrorRequest(errorLog *services.ExternalErrorLog) error
	// UpdateAnalyticsGeneralEvent upon completion of the scan and we have all the results to report on,
	// we send a finalized analytics metrics event with information matching an existing event's msi.
	UpdateAnalyticsGeneralEvent(event services.XscAnalyticsGeneralEventFinalize) error
	// GetAnalyticsGeneralEvent returns general event that match the msi provided.
	GetAnalyticsGeneralEvent(msi string) (*services.XscAnalyticsGeneralEvent, error)
	// GetConfigProfileByName returns the configuration profile that match the profile name provided.
	GetConfigProfileByName(profileName string) (*services.ConfigProfile, error)
	// GetConfigProfileByUrl returns the configuration profile related to the provided repository url.
	GetConfigProfileByUrl(profileUrl string) (*services.ConfigProfile, error)
}
