package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestSingleRequest(t *testing.T) {
	var a JWPlayerAdapter
	a.endpoint = "http://test.com/openrtb2"

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "test_placement_id"}}`),
			Video: &openrtb2.Video{
				H: 250,
				W: 350,
			},
		}},
		Site: &openrtb2.Site{},
	}

	results, err := a.MakeRequests(request, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, results, 1, "Only one request should be returned")

	result := results[0]
	resultJSON := &openrtb2.BidRequest{}
	json.Unmarshal(result.Body, resultJSON)

	expectedJSON := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "test_placement_id",
			Video: &openrtb2.Video{
				H: 250,
				W: 350,
			},
		}},
		Site: &openrtb2.Site{},
	}

	assert.Equal(t, expectedJSON, resultJSON)
}

func TestInvalidImpExt(t *testing.T) {
	var a JWPlayerAdapter
	a.endpoint = "http://test.com/openrtb2"

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	request := &openrtb2.BidRequest{
		ID: "test_id_1",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{]`),
		}},
	}

	result, err := a.MakeRequests(request, &reqInfo)

	assert.Len(t, err, 2, "2 errors should be returned")
	assert.Empty(t, result, "Result should be nil")
}

func TestIdsAreRemoved(t *testing.T) {
	var a JWPlayerAdapter
	a.endpoint = "http://test.com/openrtb2"

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "test_placement_id"}}`),
		}},
		Site: &openrtb2.Site{
			ID:     "test_site_id",
			Domain: "test_domain",
			Publisher: &openrtb2.Publisher{
				ID:   "test_publisher_id",
				Name: "testPublisher_name",
			},
		},
		App: &openrtb2.App{
			ID:     "test_app_id",
			Domain: "test_app_domain",
		},
	}

	results, err := a.MakeRequests(request, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, results, 1, "Only one request should be returned")

	result := results[0]
	resultJSON := &openrtb2.BidRequest{}
	json.Unmarshal(result.Body, resultJSON)

	assert.Len(t, resultJSON.Imp, 1, "Imp count should be equal or less than Imps from input. In this test, should be 1.")
	assert.Empty(t, resultJSON.Imp[0].Ext, "Ext should be deleted")
	assert.NotEmpty(t, resultJSON.Site, "Site object should not be removed")
	assert.Empty(t, resultJSON.Site.ID, "Site.id should be removed")
	assert.NotEmpty(t, resultJSON.Site.Publisher, "Publisher object should not be removed")
	assert.Empty(t, resultJSON.Site.Publisher.ID, "Publisher.id should be removed")
	assert.NotEmpty(t, resultJSON.App, "App object should not be removed")
	assert.Empty(t, resultJSON.App.ID, "App.id should be removed")
}

func TestOpenRTBEmptyResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}
	bidder := new(JWPlayerAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected empty response")
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusAccepted,
	}
	bidder := new(JWPlayerAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected empty response")

	assert.Equal(t, 1, len(errs), "Expected 1 error. Got %d", len(errs))
}

func TestOpenRTBStandardResponse(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Video: &openrtb2.Video{
				W: 320,
				H: 50,
			},
			Ext: json.RawMessage(`{"bidder": {
				"placementId": "2763",
			}}`),
		}},
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1234567890","impid":"test-imp-id","price": 2,"crid":"4122982","adm":"some ad","h": 50,"w": 320,"ext":{"bidder":{"appnexus":{"targeting": {"key": "rpfl_2763", "values":["43_tier0100"]},"mime": "text/html","size_id": 43}}}}]}]}`),
	}

	bidder := new(JWPlayerAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.NotNil(t, bidResponse, "Expected not empty response")
	assert.Equal(t, 1, len(bidResponse.Bids), "Expected 1 bid. Got %d", len(bidResponse.Bids))

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, openrtb_ext.BidTypeVideo, bidResponse.Bids[0].BidType,
		"Expected a video bid. Got: %s", bidResponse.Bids[0].BidType)

	theBid := bidResponse.Bids[0].Bid
	assert.Equal(t, "1234567890", theBid.ID, "Bad bid ID. Expected %s, got %s", "1234567890", theBid.ID)
}
