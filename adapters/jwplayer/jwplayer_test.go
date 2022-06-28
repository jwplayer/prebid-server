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

func getTestAdapter() adapters.Bidder {
	var mockEnricher Enricher = &MockEnricher{}
	var testAdapter adapters.Bidder = &JWPlayerAdapter{
		endpoint: "http://test.com/openrtb2",
		enricher: mockEnricher,
	}
	return testAdapter
}

type MockEnricher struct {
	Request *openrtb2.BidRequest
	SiteId  string
}

func (enricher *MockEnricher) EnrichRequest(request *openrtb2.BidRequest, siteId string) *TargetingFailed {
	enricher.Request = request
	enricher.SiteId = siteId
	return nil
}

func TestSingleRequest(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

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

	processedRequests, err := a.MakeRequests(request, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequests, 1, "Only one request should be returned")

	processedRequest := processedRequests[0]
	processedRequestJSON := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, processedRequestJSON)

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
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

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
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

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

	processedRequest, err := a.MakeRequests(request, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequest, 1, "Only one request should be returned")

	result := processedRequest[0]
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

func TestMandatoryRequestParamsAreAdded(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "test_placement_id"}}`),
		}},
	}

	processedRequests, err := a.MakeRequests(request, &reqInfo)
	assert.Empty(t, err)

	processedRequest := processedRequests[0]
	processedRequestJSON := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, processedRequestJSON)
	assert.NotNil(t, processedRequestJSON.Device)
	assert.NotNil(t, processedRequestJSON.Imp[0].Video)
}

func TestEnrichmentCall(t *testing.T) {
	enrichmentSpy := &MockEnricher{}
	var mockEnricher Enricher = enrichmentSpy
	var a adapters.Bidder = &JWPlayerAdapter{
		endpoint: "http://test.com/openrtb2",
		enricher: mockEnricher,
	}
	var reqInfo adapters.ExtraRequestInfo

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
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId","siteId": "testSiteId"}}`),
			},
		},
	}

	a.MakeRequests(request, &reqInfo)
	assert.Equal(t, "testSiteId", enrichmentSpy.SiteId)

	request = &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "test_placement_id"}}`),
			Video: &openrtb2.Video{
				H: 250,
				W: 350,
			},
		}},
		App: &openrtb2.App{
			Publisher: &openrtb2.Publisher{},
		},
	}

	a.MakeRequests(request, &reqInfo)
	assert.Empty(t, enrichmentSpy.SiteId)
}

func TestOpenRTBEmptyResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}
	bidder := getTestAdapter()
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected empty response")
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
}

func TestOpenRTBBadResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusBadRequest,
	}
	bidder := new(JWPlayerAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected empty response")
	assert.Len(t, errs, 1, "Expected 1 error. Got %d", len(errs))
}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusAccepted,
	}
	bidder := getTestAdapter()
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

	bidder := getTestAdapter()
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.NotNil(t, bidResponse, "Expected not empty response")
	assert.Equal(t, 1, len(bidResponse.Bids), "Expected 1 bid. Got %d", len(bidResponse.Bids))

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, openrtb_ext.BidTypeVideo, bidResponse.Bids[0].BidType,
		"Expected a video bid. Got: %s", bidResponse.Bids[0].BidType)

	theBid := bidResponse.Bids[0].Bid
	assert.Equal(t, "1234567890", theBid.ID, "Bad bid ID. Expected %s, got %s", "1234567890", theBid.ID)
}

func TestGetExtraInfo(t *testing.T) {
	extraInfo := getExtraInfo("{\"targeting_endpoint\": \"targetingUrl\"}")
	assert.Equal(t, "targetingUrl", extraInfo.TargetingEndpoint)

	defaultTargetingUrl := "https://content-targeting-api.longtailvideo.com/property/{{.SiteId}}/content_segments?content_url=%{{.MediaUrl}}&title={{.Title}}&description={{.Description}}"
	extraInfo = getExtraInfo("{/")
	assert.Equal(t, defaultTargetingUrl, extraInfo.TargetingEndpoint)

	extraInfo = getExtraInfo("{}")
	assert.Equal(t, defaultTargetingUrl, extraInfo.TargetingEndpoint)
}
