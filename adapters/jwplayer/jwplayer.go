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

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the JWPlayer adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  "GET",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	response.ID = request.ID

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i, _ := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			if matchingImpId := getImpId(i, *request); matchingImpId != "" {
				bid.ImpID = matchingImpId
			}

			b := &adapters.TypedBid{
				Bid:     bid,
				BidType: openrtb_ext.BidTypeVideo,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func getImpId(impIdx int, request openrtb2.BidRequest) string {
	impressions := request.Imp
	if impIdx >= len(impressions) {
		return ""
	}

	return impressions[impIdx].ID
}
