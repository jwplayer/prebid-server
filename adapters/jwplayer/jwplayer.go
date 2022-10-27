package jwplayer

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"time"
)

type Adapter struct {
	endpoint   string
	rtdAdapter RTDAdapter
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

const (
	UnexpectedStatusCodeWarning          = "Unexpected status code:"
	MissingFieldWarning                  = "The bid request is missing "
	DebugSuggestion                      = "Run with request.debug = 1 for more info."
	MissingDistributionChannelSuggestion = "Please populate either $.site or $.app."
	MissingPublisherExtSuggestion        = "$.{site|app}.publisher.ext.jwplayer.publisherId is required."
	TroubleshootingPrefix                = "We recommend populating "
)

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
	var rtdAdapter RTDAdapter
	rtdAdapter, rtdAdapterBuildError := buildContentTargeting(httpClient, extraInfo.TargetingEndpoint)

	if rtdAdapterBuildError != nil {
		fmt.Printf("Warning: a failure occured when building the RTDAdapter: %s\n", rtdAdapterBuildError)
	}

	bidder := &Adapter{
		endpoint:   config.Endpoint,
		rtdAdapter: rtdAdapter,
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

	publisherParams, invalidJwplayerPubExt := a.getJwplayerPublisherExt(publisher.Ext)
	if invalidJwplayerPubExt != nil {
		errors = append(errors, invalidJwplayerPubExt)
		return nil, errors
	}

	a.setXandrSChain(&requestCopy, publisherParams.PublisherId)

	enrichmentFailure := a.rtdAdapter.EnrichRequest(&requestCopy, publisherParams.SiteId)
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
		suggestions := a.getTroubleShootingSuggestions(request)
		return nil, suggestions
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("%s 400. Bad request from publisher. %s", UnexpectedStatusCodeWarning, DebugSuggestion),
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("%s %d. %s", UnexpectedStatusCodeWarning, responseData.StatusCode, DebugSuggestion),
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
	// Per Xandr, TagID should be the Placement Code (not to be confused with Placement ID)
	// Since we are not using Placement Codes, it is best to remove any values that may create conflicts.
	// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-ImpressionObject
	imp.TagID = ""
	// Per results obtained when testing the bid request to Xandr, imp.ext.Appnexus.placement_id is mandatory
	imp.Ext = GetXandrImpExt(placementId)
	if imp.Video == nil {
		// Per results obtained when testing the bid request to Xandr, imp.video is mandatory
		imp.Video = &openrtb2.Video{}
	}

	SetXandrVideoExt(imp.Video)

	return nil
}

func (a *Adapter) sanitizeDistributionChannels(site *openrtb2.Site, app *openrtb2.App) *errortypes.BadInput {
	if site == nil && app == nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("The bid request did not contain a Site or App field. %s", MissingDistributionChannelSuggestion),
		}
	}

	if site != nil && app != nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Per oRTB 2.5, The bid request cannot contain both a Site and App field. %s", MissingDistributionChannelSuggestion),
		}
	}

	if site != nil {
		// per Xandr doc, if set, this should equal the Xandr placement code.
		// Since we are not using Placement Codes, it is best to remove any values that may create conflicts.
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
			Message: MissingFieldWarning + "a Publisher field. " + MissingPublisherExtSuggestion,
		}
	}

	return publisher, err
}

func (a *Adapter) sanitizePublisher(publisher *openrtb2.Publisher) {
	// per Xandr doc, if set, this should equal the Xandr publisher code.
	// Used to set a default placement ID in the auction if tagid, site.id, or app.id are not provided.
	// Since we are not using Placement Codes, it is best to remove any values that may create conflicts.
	// https://docs.xandr.com/bundle/supply-partners/page/incoming-bid-request-from-ssps.html#IncomingBidRequestfromSSPs-PublisherObject
	publisher.ID = ""
}

func (a *Adapter) getJwplayerPublisherExt(pubExt json.RawMessage) (*jwplayerPublisher, *errortypes.BadInput) {
	if pubExt == nil {
		return nil, &errortypes.BadInput{
			Message: MissingFieldWarning + "publisher.ext . " + MissingPublisherExtSuggestion,
		}
	}

	var jwplayerPublisherExt publisherExt
	if err := json.Unmarshal(pubExt, &jwplayerPublisherExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Invalid publisher.ext.jwplayer in request: " + err.Error(),
		}
	}

	if jwplayerPublisherExt.JWPlayer.PublisherId == "" {
		return nil, &errortypes.BadInput{
			Message: MissingFieldWarning + "publisher.ext.jwplayer.publisherId . " + MissingPublisherExtSuggestion,
		}
	}

	return &jwplayerPublisherExt.JWPlayer, nil
}

func (a *Adapter) setXandrSChain(request *openrtb2.BidRequest, publisherId string) {
	var publisherSChain *openrtb2.SupplyChain
	if publisherSChain = GetPublisherSChain25(request.Source); publisherSChain == nil {
		publisherSChain = GetPublisherSChain26(request.Source)
	}

	// always clear 2.5  and 2.6 schain to avoid any possible confusion downstream
	if request.Source != nil {
		request.Source.SChain = nil
		request.Source.Ext = nil
	}

	sChain := MakeSChain(publisherId, request.ID, publisherSChain)
	request.Ext = GetXandrRequestExt(sChain)
}

func (a *Adapter) sanitizeRequest(request *openrtb2.BidRequest) {
	// Per results obtained when testing the bid request to Xandr, $.device is mandatory
	if request.Device == nil {
		request.Device = &openrtb2.Device{}
	}
}

func (a *Adapter) getTroubleShootingSuggestions(request *openrtb2.BidRequest) (suggestions []error) {
	if device := request.Device; device != nil {
		if device.IP == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + "$.device.ip",
				code:    TroubleShootingDeviceIPErrorCode,
			})
		}

		if device.IFA == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + "$.device.ifa",
				code:    TroubleShootingDeviceIFAErrorCode,
			})
		}
	}

	const buyerUserIdFieldName = "$.user.buyeruid"
	const userIdFieldName = "$.user.id"
	if user := request.User; user != nil {
		if user.BuyerUID == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + buyerUserIdFieldName,
				code:    TroubleShootingBuyerUIdErrorCode,
			})
		}

		if user.ID == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + userIdFieldName,
				code:    TroubleShootingUserIdErrorCode,
			})
		}
	} else {
		suggestions = append(suggestions, &Warning{
			Message: TroubleshootingPrefix + buyerUserIdFieldName + " and " + userIdFieldName,
			code:    TroubleShootingUserErrorCode,
		})
	}

	if site := request.Site; site != nil {
		if site.Ref == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + "$.site.ref",
				code:    TroubleShootingSiteRefErrorCode,
			})
		}

		if site.Domain == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + "$.site.domain",
				code:    TroubleShootingSiteDomainErrorCode,
			})
		}

		if site.Page == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + "$.site.page",
				code:    TroubleShootingSitePageErrorCode,
			})
		}
	} else if app := request.App; app != nil {
		if app.Domain == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + "$.app.domain",
				code:    TroubleShootingAppDomainErrorCode,
			})
		}

		if app.Bundle == "" {
			suggestions = append(suggestions, &Warning{
				Message: TroubleshootingPrefix + "$.app.bundle",
				code:    TroubleShootingAppBundleErrorCode,
			})
		}
	}

	return suggestions
}
