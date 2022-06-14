package jwplayer

import "github.com/prebid/prebid-server/errortypes"

const (
	MissingPublisherIdErrorCode = 301001
	MissingMediaUrlErrorCode
	EmptyTargetingSegments
	HttpRequestInstantiationErrorCode = 302000
	BaseNetworkErrorCode = 303000
	BaseDecodingErrorCode = 304000
)

type TargetingFailed struct {
	Message string
	code int
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
