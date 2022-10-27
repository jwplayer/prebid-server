package jwplayer

import "github.com/prebid/prebid-server/errortypes"

const (
	MissingDistributionChannelErrorCode = 300001
	EndpointTemplateErrorCode           = 301001
	MissingContentBlockErrorCode        = 302001
	MissingSiteIdErrorCode
	MissingMediaUrlErrorCode
	EmptyTemplateErrorCode
	TargetingUrlErrorCode
	EmptyTargetingSegmentsErrorCode
	HttpRequestInstantiationErrorCode = 303000
	HttpRequestExecutionErrorCode     = 303050
	BaseNetworkErrorCode              = 304000
	BaseDecodingErrorCode             = 305000
	TroubleShootingDeviceIPErrorCode  = 400001
	TroubleShootingDeviceIFAErrorCode
	TroubleShootingUserErrorCode
	TroubleShootingUserIdErrorCode
	TroubleShootingBuyerUIdErrorCode
	TroubleShootingSiteRefErrorCode
	TroubleShootingSiteDomainErrorCode
	TroubleShootingSitePageErrorCode
	TroubleShootingAppDomainErrorCode
	TroubleShootingAppBundleErrorCode
)

type Warning struct {
	Message string
	code    int
}

func (w *Warning) Error() string {
	return w.Message
}

func (w *Warning) Code() int {
	return w.code
}

func (w *Warning) Severity() errortypes.Severity {
	return errortypes.SeverityWarning
}
