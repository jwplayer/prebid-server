package jwplayer

import "github.com/prebid/prebid-server/errortypes"

const (
	MissingPublisherIdErrorCode = 301001
	MissingMediaUrlErrorCode
	EmptyTargetingResponse
	EmptyTargetingSegments
	BaseNetworkErrorCode = 302000
	BaseDecodingErrorCode = 303000
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
