package jwplayer

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"time"
)

type Adapter struct {
	endpoint string
	enricher Enricher
}

type ExtraInfo struct {
	TargetingEndpoint string `json:"targeting_endpoint,omitempty"`
}

type jwplayerPublisher struct {
	PublisherId string `json:"publisherId,omitempty"`
	SiteId      string `json:"siteId,omitempty"`
}

type publisherExt struct {
	JWPlayer jwplayerPublisher `json:"jwplayer,omitempty"`
}

// Builder builds a new instance of the JWPlayer adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	//configuration is consistent with default client cache config
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:     0,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     time.Duration(50) * time.Millisecond,
		},
	}

	extraInfo := ParseExtraInfo(config.ExtraAdapterInfo)
	var enricher Enricher
	enricher, enricherBuildError := buildRequestEnricher(httpClient, extraInfo.TargetingEndpoint)

	if enricherBuildError != nil {
		fmt.Printf("Warning: a failure occured when building the Enricher: %s\n", enricherBuildError)
	}

	bidder := &Adapter{
		endpoint: config.Endpoint,
		enricher: enricher,
	}

	return bidder, nil
}

func (a *Adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	requestCopy := *request
	var validImps = make([]openrtb2.Imp, 0, len(request.Imp))

	for _, imp := range requestCopy.Imp {
		params, parserError := a.parseBidderParams(imp)
		if parserError != nil {
			errors = append(errors, parserError)
		} else {
			placementId := params.PlacementId
			a.sanitizeImp(&imp, placementId)
			validImps = append(validImps, imp)
		}
	}

	if len(validImps) == 0 {
		err := &errortypes.BadInput{
			Message: "The bid request did not contain valid Imp objects.",
		}
		errors = append(errors, err)
		return nil, errors
	}

	requestCopy.Imp = validImps

	var publisherParams *jwplayerPublisher

	if site := requestCopy.Site; site != nil {
		// per Xandr doc, if set, this should equal the Xandr placement code.
		// It is best to remove, since placement code is set to imp.TagID
		// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-SiteObjectSiteObject
		site.ID = ""

		if publisher := site.Publisher; publisher != nil {
			a.sanitizePublisher(publisher)
			publisherParams = ParsePublisherParams(*publisher)
		}
	}

	if app := requestCopy.App; app != nil {
		// per Xandr doc, if set, used to look up an Xandr tinytag ID by tinytag code.
		// It is best to remove, since Xandr expects an ID specific to its platform
		// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-AppObjectAppObject
		app.ID = ""

		if publisher := app.Publisher; publisher != nil {
			a.sanitizePublisher(publisher)
			publisherParams = ParsePublisherParams(*publisher)
		}
	}

	siteId := ""
	if publisherParams != nil {
		siteId = publisherParams.SiteId
	}

	enrichmentFailure := a.enricher.EnrichRequest(&requestCopy, siteId)
	if enrichmentFailure != nil {
		errors = append(errors, enrichmentFailure)
	}
	a.sanitizeRequest(&requestCopy)

	requestJSON, err := json.Marshal(requestCopy)
	fmt.Println("Ready to make req ", string(requestJSON))
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

func (a *Adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidderResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeVideo,
			}
			bidderResponse.Bids = append(bidderResponse.Bids, b)
		}
	}

	return bidderResponse, nil
}

func (a *Adapter) parseBidderParams(imp openrtb2.Imp) (*openrtb_ext.ImpExtJWPlayer, error) {
	var impExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
		return nil, err
	}

	var params openrtb_ext.ImpExtJWPlayer
	if err := json.Unmarshal(impExt.Bidder, &params); err != nil {
		return nil, err
	}

	return &params, nil
}

func (a *Adapter) sanitizeImp(imp *openrtb2.Imp, placementId string) {
	imp.TagID = placementId
	// Per results obtained when testing the bid request to Xandr, imp.ext.Appnexus.placement_id is mandatory
	imp.Ext = GetAppnexusExt(placementId)
	if imp.Video == nil {
		// Per results obtained when testing the bid request to Xandr, imp.video is mandatory
		imp.Video = &openrtb2.Video{}
	}
}

func (a *Adapter) sanitizePublisher(publisher *openrtb2.Publisher) {
	// per Xandr doc, if set, this should equal the Xandr publisher code.
	// Used to set a default placement ID in the auction if tagid, site.id, or app.id are not provided.
	// It is best to remove, since placement code is set to imp.TagID
	// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-PublisherObject
	publisher.ID = ""
}

func (a *Adapter) sanitizeRequest(request *openrtb2.BidRequest) {
	//if request.Device == nil {
	// Per results obtained when testing the bid request to Xandr, $.device is mandatory
	request.Device = &openrtb2.Device{}
}
