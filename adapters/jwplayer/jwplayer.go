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
	enricher, enricherBuildError := buildContentTargeting(httpClient, extraInfo.TargetingEndpoint)

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

	for idx, imp := range requestCopy.Imp {
		err := a.sanitizeImp(&imp)
		if err != nil {
			err.Message = fmt.Sprintf("Imp #%d, ID %s, is invalid: %s", idx, imp.ID, err.Message)
			errors = append(errors, err)
		} else {
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

	if distributionChannelError := a.sanitizeDistributionChannels(requestCopy.Site, requestCopy.App); distributionChannelError != nil {
		errors = append(errors, distributionChannelError)
		return nil, errors
	}

	publisher, missingPublisherError := a.getPublisher(requestCopy.Site, requestCopy.App)
	if missingPublisherError != nil {
		errors = append(errors, missingPublisherError)
		return nil, errors
	}

	a.sanitizePublisher(publisher)
	publisherParams, missingJwplayerPubExt := a.getJwplayerPublisherExt(publisher.Ext)
	if missingJwplayerPubExt != nil {
		errors = append(errors, missingJwplayerPubExt)
		return nil, errors
	}

	publisherId := publisherParams.PublisherId

	if publisherId == "" {
		err := &errortypes.BadInput{
			Message: "The bid request did not contain a JWPlayer publisher Id.\n Set your Publisher Id to $.{site|app}.publisher.ext.jwplayer.publisherId.",
		}
		errors = append(errors, err)
		return nil, errors
	}

	a.setSChain(&requestCopy, publisherId)

	enrichmentFailure := a.enricher.EnrichRequest(&requestCopy, publisherParams.SiteId)
	if enrichmentFailure != nil {
		errors = append(errors, enrichmentFailure)
	}

	a.sanitizeRequest(&requestCopy)

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

func (a *Adapter) sanitizeImp(imp *openrtb2.Imp) *errortypes.BadInput {
	params, parseError := ParseBidderParams(*imp)
	if parseError != nil {
		return &errortypes.BadInput{
			Message: "Invalid Ext: " + parseError.Error(),
		}
	}

	placementId := params.PlacementId
	imp.TagID = placementId
	// Per results obtained when testing the bid request to Xandr, imp.ext.Appnexus.placement_id is mandatory
	imp.Ext = GetAppnexusExt(placementId)
	if imp.Video == nil {
		// Per results obtained when testing the bid request to Xandr, imp.video is mandatory
		imp.Video = &openrtb2.Video{}
	}

	return nil
}

func (a *Adapter) sanitizeDistributionChannels(site *openrtb2.Site, app *openrtb2.App) *errortypes.BadInput {
	if site == nil && app == nil {
		return &errortypes.BadInput{
			Message: "The bid request did not contain a Site or App field. Please populate $.{site|app}.",
		}
	}

	if site != nil && app != nil {
		return &errortypes.BadInput{
			Message: "Per oRTB 2.5, The bid request cannot contain both a Site and App field. Please populate either $.site or $.app.",
		}
	}

	if site != nil {
		// per Xandr doc, if set, this should equal the Xandr placement code.
		// It is best to remove, since placement code is set to imp.TagID
		// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-SiteObjectSiteObject
		site.ID = ""

	}

	if app != nil {
		// per Xandr doc, if set, used to look up an Xandr tinytag ID by tinytag code.
		// It is best to remove, since Xandr expects an ID specific to its platform
		// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-AppObjectAppObject
		app.ID = ""
	}

	return nil
}

func (a *Adapter) getPublisher(site *openrtb2.Site, app *openrtb2.App) (publisher *openrtb2.Publisher, err *errortypes.BadInput) {
	if site != nil {
		publisher = site.Publisher
	} else if app != nil {
		publisher = app.Publisher
	}

	if publisher == nil {
		err = &errortypes.BadInput{
			Message: "The bid request did not contain a Publisher field. Please populate $.{site|app}.publisher .",
		}
	}

	return publisher, err
}

func (a *Adapter) sanitizePublisher(publisher *openrtb2.Publisher) {
	// per Xandr doc, if set, this should equal the Xandr publisher code.
	// Used to set a default placement ID in the auction if tagid, site.id, or app.id are not provided.
	// It is best to remove, since placement code is set to imp.TagID
	// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-PublisherObject
	publisher.ID = ""
}

func (a *Adapter) getJwplayerPublisherExt(pubExt json.RawMessage) (*jwplayerPublisher, *errortypes.BadInput) {
	if pubExt == nil {
		return nil, &errortypes.BadInput{
			Message: "The bid request is missing publisher.ext.\n $.{site|app}.publisher.ext.jwplayer.publisherId is required.",
		}
	}

	var jwplayerPublisherExt publisherExt
	if err := json.Unmarshal(pubExt, &jwplayerPublisherExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Invalid publisher.ext.jwplayer in request: " + err.Error(),
		}
	}

	if &jwplayerPublisherExt.JWPlayer == nil {
		return nil, &errortypes.BadInput{
			Message: "The bid request is missing publisher.ext.jwplayer\n $.{site|app}.publisher.ext.jwplayer.publisherId is required.",
		}
	}

	return &jwplayerPublisherExt.JWPlayer, nil
}

func (a *Adapter) setSChain(request *openrtb2.BidRequest, publisherId string) {
	publisherSChain := GetPublisherSChain25(request.Source)
	a.clearPublisherSChain(request.Source)
	sChain := MakeSChain(publisherId, request.ID, publisherSChain)
	request.Ext = GetXandrRequestExt(sChain)
}

func (a *Adapter) clearPublisherSChain(source *openrtb2.Source) {
	if source == nil {
		return
	}
	source.Ext = nil
}

func (a *Adapter) sanitizeRequest(request *openrtb2.BidRequest) {
	// Per results obtained when testing the bid request to Xandr, $.device is mandatory
	if request.Device == nil {
		request.Device = &openrtb2.Device{}
	}
}
