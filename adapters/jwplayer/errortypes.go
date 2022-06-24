package jwplayer

import "github.com/prebid/prebid-server/errortypes"

const (
	EndpointTemplateErrorCode = 301001
	MissingSiteIdErrorCode    = 302001
	MissingMediaUrlErrorCode
	EmptyTargetingSegments
	MacroResolveErrorCode
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
