package jwplayer

import "github.com/prebid/prebid-server/v3/errortypes"

const (
	MissingDistributionChannelErrorCode = 300001
	EndpointTemplateErrorCode           = 301001
	MissingContentBlockErrorCode        = iota + 302001
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
