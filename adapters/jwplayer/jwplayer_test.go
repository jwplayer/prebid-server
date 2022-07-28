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
	var mockRtdAdapter RTDAdapter = &MockRTDAdapter{}
	var testAdapter adapters.Bidder = &Adapter{
		endpoint:   "http://test.com/openrtb2",
		rtdAdapter: mockRtdAdapter,
	}
	return testAdapter
}

type MockRTDAdapter struct {
	Request *openrtb2.BidRequest
	SiteId  string
}

func (rtdAdapter *MockRTDAdapter) EnrichRequest(request *openrtb2.BidRequest, siteId string) EnrichmentFailed {
	rtdAdapter.Request = request
	rtdAdapter.SiteId = siteId
	return nil
}

func TestSingleRequest(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	rawRequest := &openrtb2.BidRequest{
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

	processedRequests, err := a.MakeRequests(rawRequest, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequests, 1, "Only one request should be returned")

	processedRequest := processedRequests[0]
	bidRequest := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, bidRequest)

	expectedJSON := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{
				H: 250,
				W: 350,
			},
			Ext: json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}

	assert.Equal(t, expectedJSON, bidRequest)
}

func TestInvalidImpExt(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	rawRequest := &openrtb2.BidRequest{
		ID: "test_id_1",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{]`),
		}},
	}

	result, err := a.MakeRequests(rawRequest, &reqInfo)

	assert.Len(t, err, 2, "2 errors should be returned")
	assert.Empty(t, result, "Result should be nil")
}

func TestInvalidImpAreFiltered(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	rawRequest := &openrtb2.BidRequest{
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
				Ext: json.RawMessage(`{"jwplayer":{"publisherId":"testPublisherId"}}`),
			},
		},
	}

	processedRequests, err := a.MakeRequests(rawRequest, &reqInfo)

	assert.Len(t, err, 2, "2 errors should be returned")
	assert.NotNil(t, processedRequests, "Result should be valid. Some impressions are valid")
	assert.Len(t, processedRequests, 1, "Only one request should be returned")

	processedRequest := processedRequests[0]
	processedRequestJSON := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, processedRequestJSON)

	expectedRequest := &openrtb2.BidRequest{
		ID: "test_id_1",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id_valid",
			Video: &openrtb2.Video{},
			TagID: "1",
			Ext:   json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}, {
			ID:    "test_imp_id_valid_2",
			Video: &openrtb2.Video{},
			TagID: "2",
			Ext:   json.RawMessage(`{"appnexus":{"placement_id":2}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage(`{"jwplayer":{"publisherId":"testPublisherId"}}`),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id_1","hp":1}],"ver":"1.0"}}`)),
	}

	assert.Equal(t, expectedRequest, processedRequestJSON)
}

func TestImpVideoExt(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	rawRequest := &openrtb2.BidRequest{
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

	processedRequests, err := a.MakeRequests(rawRequest, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequests, 1, "Only one request should be returned")

	processedRequest := processedRequests[0]
	bidRequest := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, bidRequest)

	expectedRequest := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{
				H: 250,
				W: 350,
			},
			Ext: json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}

	assert.Equal(t, expectedRequest, bidRequest)

	rawRequest.Imp[0].Video.Placement = openrtb2.VideoPlacementTypeInFeed
	processedRequests, err = a.MakeRequests(rawRequest, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequests, 1, "Only one request should be returned")

	processedRequest = processedRequests[0]
	json.Unmarshal(processedRequest.Body, bidRequest)

	expectedRequest = &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{
				H:         250,
				W:         350,
				Placement: openrtb2.VideoPlacementTypeInFeed,
				Ext:       json.RawMessage(`{"appnexus":{"context":4}}`),
			},
			Ext: json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}

	assert.Equal(t, expectedRequest, bidRequest)

	rawRequest.Imp[0].Video.Placement = openrtb2.VideoPlacementTypeInStream
	rawRequest.Imp[0].Video.StartDelay = openrtb2.StartDelay(10).Ptr()
	processedRequests, err = a.MakeRequests(rawRequest, &reqInfo)

	processedRequest = processedRequests[0]
	json.Unmarshal(processedRequest.Body, bidRequest)

	expectedRequest = &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{
				H:          250,
				W:          350,
				Placement:  openrtb2.VideoPlacementTypeInStream,
				StartDelay: openrtb2.StartDelay(10).Ptr(),
				Ext:        json.RawMessage(`{"appnexus":{"context":2}}`),
			},
			Ext: json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}

	assert.Equal(t, expectedRequest, bidRequest)
}

func TestIdsAreRemoved(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	rawRequest := &openrtb2.BidRequest{
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

	processedRequests, err := a.MakeRequests(rawRequest, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequests, 1, "Only one request should be returned")

	request := processedRequests[0]
	bidRequest := &openrtb2.BidRequest{}
	json.Unmarshal(request.Body, bidRequest)

	expectedRequest := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{},
			Ext:   json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		Site: &openrtb2.Site{
			Domain: "test_domain",
			Publisher: &openrtb2.Publisher{
				Name: "testPublisher_name",
				Ext:  json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}

	assert.Equal(t, expectedRequest, bidRequest)

	rawRequest = &openrtb2.BidRequest{
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

	processedRequests, err = a.MakeRequests(rawRequest, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, processedRequests, 1, "Only one request should be returned")

	request = processedRequests[0]
	bidRequest = &openrtb2.BidRequest{}
	json.Unmarshal(request.Body, bidRequest)

	expectedRequest = &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{},
			Ext:   json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		App: &openrtb2.App{
			Domain: "test_app_domain",
			Publisher: &openrtb2.Publisher{
				Name: "testPublisher_name",
				Ext:  json.RawMessage(`{"jwplayer":{"publisherId":"testPublisherId"}}`),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}

	assert.Equal(t, expectedRequest, bidRequest)
}

func TestMandatoryRequestParamsAreAdded(t *testing.T) {
	a := getTestAdapter()
	var reqInfo adapters.ExtraRequestInfo

	rawRequest := &openrtb2.BidRequest{
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

	processedRequests, err := a.MakeRequests(rawRequest, &reqInfo)
	assert.Empty(t, err)

	processedRequest := processedRequests[0]
	bidRequest := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, bidRequest)

	expectedRequest := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{},
			Ext:   json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}
	assert.Equal(t, expectedRequest, bidRequest)
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

	rawRequest := &openrtb2.BidRequest{
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

	processedRequests, err := a.MakeRequests(rawRequest, &reqInfo)
	assert.Empty(t, err)

	processedRequest := processedRequests[0]
	bidRequest := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, bidRequest)

	expectedRequest := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{},
			Ext:   json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}
	assert.Equal(t, expectedRequest, bidRequest)
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

	rawRequest := &openrtb2.BidRequest{
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

	processedRequests, err := a.MakeRequests(rawRequest, &reqInfo)
	assert.Empty(t, err)

	processedRequest := processedRequests[0]
	bidRequest := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, bidRequest)

	expectedRequest := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "2",
			Video: &openrtb2.Video{},
			Ext:   json.RawMessage(`{"appnexus":{"placement_id":2}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Source: &openrtb2.Source{},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":0,"nodes":[{"asi":"publisher.com","sid":"some id","rid":"some req id","hp":0},{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}

	assert.Equal(t, expectedRequest, bidRequest)
}

func TestEnrichmentCall(t *testing.T) {
	enrichmentSpy := &MockRTDAdapter{}
	var mockRtdAdapter RTDAdapter = enrichmentSpy
	var a adapters.Bidder = &Adapter{
		endpoint:   "http://test.com/openrtb2",
		rtdAdapter: mockRtdAdapter,
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
	bidRequest := &openrtb2.BidRequest{}
	json.Unmarshal(processedRequest.Body, bidRequest)

	expectedRequest := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:    "test_imp_id",
			TagID: "1",
			Video: &openrtb2.Video{},
			Ext:   json.RawMessage(`{"appnexus":{"placement_id":1}}`),
		}},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				Ext: json.RawMessage((`{"jwplayer":{"publisherId":"testPublisherId"}}`)),
			},
		},
		Source: &openrtb2.Source{},
		Device: &openrtb2.Device{},
		Ext:    json.RawMessage((`{"schain":{"complete":1,"nodes":[{"asi":"jwplayer.com","sid":"testPublisherId","rid":"test_id","hp":1}],"ver":"1.0"}}`)),
	}
	assert.Equal(t, expectedRequest, bidRequest)
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
