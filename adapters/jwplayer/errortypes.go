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
)

type TargetingFailed struct {
	Message string
	code    int
}

func (err *TargetingFailed) Error() string {
	return err.Message
}

func (err *TargetingFailed) Code() int {
	return err.code
}

func (err *TargetingFailed) Severity() errortypes.Severity {
	return errortypes.SeverityWarning
}
