package jwplayer

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func getTestAdapter() adapters.Bidder {
	var mockEnricher Enricher = &MockEnricher{}
	var testAdapter adapters.Bidder = &Adapter{
		endpoint: "http://test.com/openrtb2",
		enricher: mockEnricher,
	}
	return testAdapter
}

type MockEnricher struct {
	Request *openrtb2.BidRequest
	SiteId  string
}

func (enricher *MockEnricher) EnrichRequest(request *openrtb2.BidRequest, siteId string) EnrichmentFailed {
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
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
			Video: &openrtb2.Video{
				H: 250,
				W: 350,
			},
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
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
			Ext: json.RawMessage(`{appnexus:{placement_id:1}}`),
		}},
		Site:   &openrtb2.Site{},
		Device: &openrtb2.Device{},
	}

	assert.Equal(t, expectedJSON, processedRequestJSON)
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

func TestInvalidImpAreFiltered(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id_1",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id_valid",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}, {
			ID:  "test_imp_id_bad_format",
			Ext: json.RawMessage(`{]`),
		}, {
			ID:  "test_imp_id_valid_2",
			Ext: json.RawMessage(`{"bidder":{"placementId": "2"}}`),
		}, {
			ID:  "test_imp_id_bad_missing_placementId",
			Ext: json.RawMessage(`{"bidder":{"other": "otherId"}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
	}

	processedRequests, err := a.MakeRequests(request, &reqInfo)

	assert.Len(t, err, 2, "2 errors should be returned")
	assert.NotNil(t, processedRequests, "Result should be nil")
	assert.Len(t, processedRequests, 1, "Only one request should be returned")

	processedRequest := processedRequests[0]
	processedRequestJSON := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, processedRequestJSON)

	assert.Len(t, processedRequestJSON.Imp, 2, "Imp count should be equal or less than Imps from input. In this test, should be 2.")
	assert.Equal(t, "1", processedRequestJSON.Imp[0].TagID, "placement id should be set to TagID")
	assert.Equal(t, "2", processedRequestJSON.Imp[1].TagID, "placement id should be set to TagID")
	assert.NotNil(t, processedRequestJSON.Imp[0].Video, "Video should be populated")
	assert.NotNil(t, processedRequestJSON.Imp[1].Video, "Video should be populated")

	assert.NotNil(t, processedRequestJSON.Imp[0].Ext, "Ext should be deleted")
	assert.NotNil(t, processedRequestJSON.Imp[1].Ext, "Ext should be deleted")

	ext1 := &appnexusImpExt{}
	json.Unmarshal(processedRequestJSON.Imp[0].Ext, ext1)
	assert.Equal(t, 1, ext1.Appnexus.PlacementID)

	ext2 := &appnexusImpExt{}
	json.Unmarshal(processedRequestJSON.Imp[1].Ext, ext2)
	assert.Equal(t, 2, ext2.Appnexus.PlacementID)
}

func TestIdsAreRemoved(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		Site: &openrtb2.Site{
			ID:     "test_site_id",
			Domain: "test_domain",
			Publisher: &openrtb2.Publisher{
				ID:   "test_publisher_id",
				Name: "testPublisher_name",
				Ext:  json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
	}

	processedRequest, err := a.MakeRequests(request, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequest, 1, "Only one request should be returned")

	result := processedRequest[0]
	resultJSON := &openrtb2.BidRequest{}
	json.Unmarshal(result.Body, resultJSON)

	assert.Len(t, resultJSON.Imp, 1, "Imp count should be equal or less than Imps from input. In this test, should be 1.")
	assert.NotNil(t, resultJSON.Imp[0].Ext, "Ext should be set")
	assert.NotEmpty(t, resultJSON.Site, "Site object should not be removed")
	assert.Empty(t, resultJSON.Site.ID, "Site.id should be removed")
	assert.NotEmpty(t, resultJSON.Site.Publisher, "Publisher object should not be removed")
	assert.Empty(t, resultJSON.Site.Publisher.ID, "Publisher.id should be removed")

	request = &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		App: &openrtb2.App{
			ID:     "test_app_id",
			Domain: "test_app_domain",
			Publisher: &openrtb2.Publisher{
				ID:   "test_publisher_id",
				Name: "testPublisher_name",
				Ext:  json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
	}

	processedRequest, err = a.MakeRequests(request, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequest, 1, "Only one request should be returned")

	result = processedRequest[0]
	resultJSON = &openrtb2.BidRequest{}
	json.Unmarshal(result.Body, resultJSON)

	assert.Len(t, resultJSON.Imp, 1, "Imp count should be equal or less than Imps from input. In this test, should be 1.")
	assert.NotNil(t, resultJSON.Imp[0].Ext, "Ext should be set")
	assert.NotEmpty(t, resultJSON.App, "App object should not be removed")
	assert.Empty(t, resultJSON.App.ID, "App.id should be removed")
	assert.NotEmpty(t, resultJSON.App.Publisher, "Publisher object should not be removed")
	assert.Empty(t, resultJSON.App.Publisher.ID, "Publisher.id should be removed")
}

func TestMandatoryRequestParamsAreAdded(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
	}

	processedRequests, err := a.MakeRequests(request, &reqInfo)
	assert.Empty(t, err)

	processedRequest := processedRequests[0]
	processedRequestJSON := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, processedRequestJSON)
	assert.NotNil(t, processedRequestJSON.Device)
	assert.NotNil(t, processedRequestJSON.Imp[0].Video)
}

func TestBadInputMissingDistributionChannel(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
	}

	_, err := a.MakeRequests(request, &reqInfo)
	assert.Len(t, err, 1)
	assert.Equal(t, fmt.Sprintf("%T", &errortypes.BadInput{}), fmt.Sprintf("%T", err[0]))
}

func TestBadInputMissingPublisher(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		Site: &openrtb2.Site{
			ID: "some_id",
		},
	}

	_, err := a.MakeRequests(request, &reqInfo)
	assert.Len(t, err, 1)
	assert.Equal(t, fmt.Sprintf("%T", &errortypes.BadInput{}), fmt.Sprintf("%T", err[0]))
}

func TestBadInputMissingPublisherExt(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "some_pub_id",
			},
		},
	}

	_, err := a.MakeRequests(request, &reqInfo)
	assert.Len(t, err, 1)
	assert.Equal(t, fmt.Sprintf("%T", &errortypes.BadInput{}), fmt.Sprintf("%T", err[0]))
}

func TestBadInputMissingJwplayerPublisherExt(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"bidder":{"siteId": "testSiteId"}}`),
			},
		},
	}

	_, err := a.MakeRequests(request, &reqInfo)
	assert.Len(t, err, 1)
	assert.Equal(t, fmt.Sprintf("%T", &errortypes.BadInput{}), fmt.Sprintf("%T", err[0]))
}

func TestBadInputMissingJwplayerPublisherId(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"siteId": "testSiteId"}}`),
			},
		},
	}

	_, err := a.MakeRequests(request, &reqInfo)
	assert.Len(t, err, 1)
	assert.Equal(t, fmt.Sprintf("%T", &errortypes.BadInput{}), fmt.Sprintf("%T", err[0]))
}

func TestSChain(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
	}

	processedRequests, err := a.MakeRequests(request, &reqInfo)
	assert.Empty(t, err)

	processedRequest := processedRequests[0]
	processedRequestJSON := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, processedRequestJSON)
	assert.NotNil(t, processedRequestJSON.Ext)
	var requestExtJSON requestExt
	parseErr := json.Unmarshal(processedRequestJSON.Ext, &requestExtJSON)
	assert.Nil(t, parseErr)
	assert.NotNil(t, requestExtJSON.SChain)
	sChain := requestExtJSON.SChain
	assert.Equal(t, 1, sChain.Complete)
	assert.Equal(t, "1.0", sChain.Ver)
	assert.Len(t, sChain.Nodes, 1)
	node := sChain.Nodes[0]
	assert.Equal(t, jwplayerDomain, node.ASI)
	assert.Equal(t, "testPublisherId", node.SID)
	assert.Equal(t, "test_id", node.RID)
	assert.Equal(t, 1, node.HP)
}

func TestAppendingToExistingSchain(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	sourceExt := &openrtb_ext.SourceExt{
		SChain: openrtb_ext.ExtRequestPrebidSChainSChain{
			Complete: 0,
			Ver:      "2.0",
			Nodes: []*openrtb_ext.ExtRequestPrebidSChainSChainNode{{
				ASI: "publisher.com",
				SID: "some id",
				RID: "some req id",
				HP:  0,
			}},
		},
	}

	sourceExtJSON, _ := json.Marshal(sourceExt)

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "2"}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
		Source: &openrtb2.Source{
			Ext: sourceExtJSON,
		},
	}

	processedRequests, err := a.MakeRequests(request, &reqInfo)
	assert.Empty(t, err)

	processedRequest := processedRequests[0]
	processedRequestJSON := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, processedRequestJSON)
	assert.NotNil(t, processedRequestJSON.Ext)

	var requestExtJSON requestExt
	parseErr := json.Unmarshal(processedRequestJSON.Ext, &requestExtJSON)
	assert.Nil(t, parseErr)
	assert.NotNil(t, requestExtJSON.SChain)
	sChain := requestExtJSON.SChain
	assert.Equal(t, 0, sChain.Complete)
	assert.Equal(t, "1.0", sChain.Ver)
	assert.Len(t, sChain.Nodes, 2)

	publisherNode := sChain.Nodes[0]
	assert.Equal(t, "publisher.com", publisherNode.ASI)
	assert.Equal(t, "some id", publisherNode.SID)
	assert.Equal(t, "some req id", publisherNode.RID)
	assert.Equal(t, 0, publisherNode.HP)

	jwplayerNode := sChain.Nodes[1]
	assert.Equal(t, jwplayerDomain, jwplayerNode.ASI)
	assert.Equal(t, "testPublisherId", jwplayerNode.SID)
	assert.Equal(t, "test_id", jwplayerNode.RID)
	assert.Equal(t, 1, jwplayerNode.HP)
}

func TestEnrichmentCall(t *testing.T) {
	enrichmentSpy := &MockEnricher{}
	var mockEnricher Enricher = enrichmentSpy
	var a adapters.Bidder = &Adapter{
		endpoint: "http://test.com/openrtb2",
		enricher: mockEnricher,
	}
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "2"}}`),
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
			Ext: json.RawMessage(`{"bidder":{"placementId": "3"}}`),
			Video: &openrtb2.Video{
				H: 250,
				W: 350,
			},
		}},
		App: &openrtb2.App{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
	}

	a.MakeRequests(request, &reqInfo)
	assert.Empty(t, enrichmentSpy.SiteId)
}

func TestSourceSanitization(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	request := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
		Source: &openrtb2.Source{
			Ext: json.RawMessage(`{}`),
		},
	}

	processedRequests, err := a.MakeRequests(request, &reqInfo)
	assert.Empty(t, err)

	processedRequest := processedRequests[0]
	processedRequestJSON := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, processedRequestJSON)
	assert.NotNil(t, processedRequestJSON.Source)
	assert.Empty(t, processedRequestJSON.Source.Ext)
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
	bidder := new(Adapter)
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
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId": "testPublisherId"}}`),
			},
		},
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
