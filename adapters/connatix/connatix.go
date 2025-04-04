package connatix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	maxImpsPerRequest = 1
)

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if request.Device == nil || (request.Device.IP == "" && request.Device.IPv6 == "") {
		return nil, []error{&errortypes.BadInput{
			Message: "Device IP is required",
		}}
	}

	// connatix adapter expects imp.displaymanagerver to be populated in openrtb2 request
	// but some SDKs will put it in imp.ext.prebid instead
	displayManagerVer := buildDisplayManagerVer(request)
	var errs []error
	var validImps []openrtb2.Imp

	for _, imp := range request.Imp {
		impCopy := imp
		impExt, err := validateAndBuildImpExt(&impCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := buildRequestImp(&impCopy, impExt, displayManagerVer, reqInfo); err != nil {
			errs = append(errs, err)
			continue
		}

		validImps = append(validImps, impCopy)
	}

	requests, splitErrs := splitRequests(validImps, request, a.endpoint)
	return requests, append(errs, splitErrs...)
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var connatixResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &connatixResponse); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for _, sb := range connatixResponse.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]
			var bidExt bidExt
			var bidType openrtb_ext.BidType

			if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
				bidType = openrtb_ext.BidTypeBanner
			} else {
				bidType = getBidType(bidExt)
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	bidderResponse.Currency = "USD"
	return bidderResponse, nil
}

func validateAndBuildImpExt(imp *openrtb2.Imp) (impExtIncoming, error) {
	var ext impExtIncoming

	bidderJSON, _, _, err := jsonparser.Get(imp.Ext, "bidder")
	if err != nil {
		return impExtIncoming{}, &errortypes.BadInput{
			Message: "Missing 'bidder' object",
		}
	}

	if placementId, err := jsonparser.GetString(bidderJSON, "placementId"); err == nil {
		ext.Bidder.PlacementId = placementId
	} else {
		return impExtIncoming{}, &errortypes.BadInput{
			Message: "Invalid Placement ID",
		}
	}

	if viewability, err := jsonparser.GetFloat(bidderJSON, "viewabilityPercentage"); err == nil {
		ext.Bidder.ViewabilityPercentage = viewability
	}

	return ext, nil
}

func splitRequests(imps []openrtb2.Imp, originalRequest *openrtb2.BidRequest, uri string) ([]*adapters.RequestData, []error) {
	var errs []error
	// Initial capacity for future array of requests, memory optimization.
	// Let's say there are 35 impressions and limit impressions per request equals to 10.
	// In this case we need to create 4 requests with 10, 10, 10 and 5 impressions.
	// With this formula initial capacity=(35+10-1)/10 = 4

	var requests []*adapters.RequestData

	if len(imps) == 0 {
		return nil, nil
	}

	baseEndpoint, err := url.Parse(uri)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}

	if originalRequest.Device != nil {
		if ua := originalRequest.Device.UA; ua != "" {
			headers.Add("User-Agent", ua)
		}

		if ip := originalRequest.Device.IPv6; ip != "" {
			headers.Add("X-Forwarded-For", ip)
		}

		if ip := originalRequest.Device.IP; ip != "" {
			headers.Add("X-Forwarded-For", ip)
		}
	}

	endpoint := *baseEndpoint
	if originalRequest.User != nil {
		userID := strings.TrimSpace(originalRequest.User.BuyerUID)
		if userID != "" {
			queryParams := url.Values{}
			switch {
			case strings.HasPrefix(userID, "1-"):
				queryParams.Add("dc", "us-east-2")
			case strings.HasPrefix(userID, "2-"):
				queryParams.Add("dc", "us-west-2")
			case strings.HasPrefix(userID, "3-"):
				queryParams.Add("dc", "eu-west-1")
			}
			endpoint.RawQuery = queryParams.Encode()
		}
	}

	for start := 0; start < len(imps); start += maxImpsPerRequest {
		end := start + maxImpsPerRequest
		if end > len(imps) {
			end = len(imps)
		}
		impsForRequest := imps[start:end]
		requestCopy := *originalRequest
		requestCopy.Imp = impsForRequest

		requestJSON, err := jsonutil.Marshal(&requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     endpoint.String(),
			Body:    requestJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(impsForRequest),
		})
	}

	return requests, errs
}

func buildRequestImp(imp *openrtb2.Imp, ext impExtIncoming, displayManagerVer string, reqInfo *adapters.ExtraRequestInfo) error {
	if imp.Banner != nil {
		bannerCopy := *imp.Banner

		if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
			firstFormat := bannerCopy.Format[0]
			bannerCopy.W = &(firstFormat.W)
			bannerCopy.H = &(firstFormat.H)
		}
		imp.Banner = &bannerCopy
	}

	// Populate imp.displaymanagerver if the client failed to do it.
	if imp.DisplayManagerVer == "" && displayManagerVer != "" {
		imp.DisplayManagerVer = displayManagerVer
	}

	// Check if imp comes with bid floor amount defined in a foreign currency
	if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, "USD") {
		// Convert to US dollars
		convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
		if err != nil {
			return err
		}

		// Update after conversion. All imp elements inside request.Imp are shallow copies
		// therefore, their non-pointer values are not shared memory and are safe to modify.
		imp.BidFloor = convertedValue
		imp.BidFloorCur = "USD"
	}

	impExt := impExt{
		Connatix: impExtConnatix{
			PlacementId:           ext.Bidder.PlacementId,
			ViewabilityPercentage: ext.Bidder.ViewabilityPercentage,
		},
	}

	var err error
	imp.Ext, err = json.Marshal(impExt)
	return err
}

func buildDisplayManagerVer(req *openrtb2.BidRequest) string {
	if req.App == nil {
		return ""
	}

	source, err := jsonparser.GetString(req.App.Ext, openrtb_ext.PrebidExtKey, "source")
	if err != nil {
		return ""
	}

	version, err := jsonparser.GetString(req.App.Ext, openrtb_ext.PrebidExtKey, "version")
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s-%s", source, version)
}

func getBidType(ext bidExt) openrtb_ext.BidType {
	if ext.Cnx.MediaType == "video" {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}
