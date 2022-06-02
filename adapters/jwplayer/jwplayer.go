package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"fmt"
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
	var errors []error
	requestCopy := *request
	for _, imp := range requestCopy.Imp {
		params, parserErrors := parseBidderParams(imp)
		if parserErrors != nil {
			errors = append(errors, parserErrors...)
		} else {
			placementId := params.PlacementId
			imp.TagID = placementId
		}

		imp.Ext = nil
	}

	if site := requestCopy.Site; site != nil {
		// per Xandr doc, if set, this should equal the Xandr placement code.
		// It is best to remove, since placement code is set to imp.TagID
		// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-SiteObjectSiteObject
		site.ID = ""

		if publisher := site.Publisher; publisher != nil {
			// per Xandr doc, if set, this should equal the Xandr publisher code.
			// Used to set a default placement ID in the auction if tagid, site.id, or app.id are not provided.
			// It is best to remove, since placement code is set to imp.TagID
			// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-PublisherObject
			publisher.ID = ""
		}
	}

	if app := requestCopy.App; app != nil {
		// per Xandr doc, if set, used to look up an Xandr tinytag ID by tinytag code.
		// It is best to remove, since Xandr expects an ID specific to its platform
		// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-AppObjectAppObject
		app.ID = ""
	}

	requestJSON, err := json.Marshal(requestCopy)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
	}

	return []*adapters.RequestData{requestData}, errors
}

func (a *JWPlayerAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	fmt.Println("Make Bids")
	return nil, nil
}

func parseBidderParams(imp openrtb2.Imp) (*openrtb_ext.ImpExtJWPlayer, []error) {
	var errors []error
	var impExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	var params openrtb_ext.ImpExtJWPlayer
	if err := json.Unmarshal(impExt.Bidder, &params); err != nil {
		errors = append(errors, err)
	}

	return &params, errors
}
