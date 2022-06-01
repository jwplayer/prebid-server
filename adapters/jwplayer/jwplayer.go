package jwplayer

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type JWPlayerAdapter struct {
	endpoint string
}

// Builder builds a new instance of the JWPlayer adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &JWPlayerAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *JWPlayerAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
    requestCopy := *request
    for _,imp:= range requestCopy.Imp {
        placementId := imp.ext.prebid.bidder.jwplayer.placementId
        imp.tagid = placementId
    }

    if site := requestCopy.site; site != nil {
        delete(site, "id")

        if publisher := site.publisher; publisher != nil {
            delete(publisher, "id")
        }
    }

    if app := requestCopy.app; app != nil {
        delete(app, "id")
    }


}
