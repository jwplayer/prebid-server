package jwplayer

import (
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
)

type RTDAdapter interface {
	EnrichRequest(request *openrtb2.BidRequest, siteId string) *errortypes.TroubleShootingSuggestion
}
