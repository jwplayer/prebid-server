package jwplayer

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
)

type RTDAdapter interface {
	EnrichRequest(request *openrtb2.BidRequest, siteId string) *errortypes.TroubleShootingSuggestion
}
