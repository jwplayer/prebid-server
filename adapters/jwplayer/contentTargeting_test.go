package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Missing Tests: HttpRequestInstantiationErrorCode HttpRequestExecutionErrorCode

func TestSuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(http.StatusOK)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2", "3", "4"], "targeting_profiles": ["5", "6", "7", "8"]}}`))
	}))
	defer server.Close()

	enricher, failure := buildContentTargeting(server.Client(), server.URL)

	assert.Empty(t, failure)

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
		App: &openrtb2.App{
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	enrichmentFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Empty(t, enrichmentFailure)

	expectedEnrichedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Data: []openrtb2.Data{{
					Name:    "jwplayer.com",
					Segment: []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "3"}, {Value: "4"}, {Value: "5"}, {Value: "6"}, {Value: "7"}, {Value: "8"}},
					Ext:     json.RawMessage(`{"segtax":502}`),
				}},
				Ext: json.RawMessage(`{"description"": "testDesc"`),
			},
			Keywords: "jwpseg=1,jwpseg=2,jwpseg=3,jwpseg=4,jwpseg=5,jwpseg=6,jwpseg=7,jwpseg=8",
		},
	}
	assert.Equal(t, expectedEnrichedRequest, request)
}

func TestSuccessfulAppendsToKeywords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(200)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2"], "targeting_profiles": ["5", "6"]}}`))
	}))
	defer server.Close()
	enricher, _ := buildContentTargeting(server.Client(), server.URL)

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
			Keywords: "existingKey=value",
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Keywords: "existingKey=value,jwpseg=1,jwpseg=2,jwpseg=5,jwpseg=6",
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Data: []openrtb2.Data{{
					Name:    "jwplayer.com",
					Segment: []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "5"}, {Value: "6"}},
					Ext:     json.RawMessage(`{"segtax":502}`),
				}},
				Ext: json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}
	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Empty(t, targetingFailure)
	assert.Equal(t, expectedRequest, request)
}

func TestSuccessAppendsToPreviousData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(200)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2"], "targeting_profiles": ["5", "6"]}}`))
	}))
	defer server.Close()
	enricher, _ := buildContentTargeting(server.Client(), server.URL)

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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Data: []openrtb2.Data{{
					Name: "otherData",
					ID:   "otherDataId",
				}, {
					Name: "3rdData",
					ID:   "3rdDataId",
				}},
				Ext: json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Data: []openrtb2.Data{{
					Name: "otherData",
					ID:   "otherDataId",
				}, {
					Name: "3rdData",
					ID:   "3rdDataId",
				}, {
					Name:    "jwplayer.com",
					Segment: []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "5"}, {Value: "6"}},
					Ext:     json.RawMessage(`{"segtax":502}`),
				}},
				Ext: json.RawMessage(`{"description"": "testDesc"`),
			},
			Keywords: "jwpseg=1,jwpseg=2,jwpseg=5,jwpseg=6",
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Empty(t, targetingFailure)
	assert.Equal(t, expectedRequest, request)
}

func TestMissingDistributionChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(http.StatusOK)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2", "3", "4"], "targeting_profiles": ["5", "6", "7", "8"]}}`))
	}))
	defer server.Close()

	enricher, failure := buildContentTargeting(server.Client(), server.URL)

	assert.Empty(t, failure)

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
	}

	enrichmentFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Equal(t, MissingDistributionChannelErrorCode, enrichmentFailure.Code())
}

func TestMissingContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(http.StatusOK)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2", "3", "4"], "targeting_profiles": ["5", "6", "7", "8"]}}`))
	}))
	defer server.Close()

	enricher, failure := buildContentTargeting(server.Client(), server.URL)

	assert.Empty(t, failure)

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
		App: &openrtb2.App{},
	}

	expectedRequest := &openrtb2.BidRequest{
		ID: "test_id",
		Imp: []openrtb2.Imp{{
			ID:  "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "test_placement_id"}}`),
			Video: &openrtb2.Video{
				H: 250,
				W: 350,
			},
		}},
		App: &openrtb2.App{},
	}

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Equal(t, MissingContentBlockErrorCode, targetingFailure.Code())
	assert.Equal(t, expectedRequest, request)
}

func TestDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(http.StatusOK)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2", `))
	}))
	defer server.Close()

	enricher, failure := buildContentTargeting(server.Client(), server.URL)

	assert.Empty(t, failure)

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
		App: &openrtb2.App{
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Equal(t, expectedRequest, request)
	assert.Equal(t, BaseDecodingErrorCode, targetingFailure.Code())
}

func TestNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(433)
	}))
	defer server.Close()

	enricher, failure := buildContentTargeting(server.Client(), server.URL)

	assert.Empty(t, failure)

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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Equal(t, expectedRequest, request)
	assert.Equal(t, BaseNetworkErrorCode+433, targetingFailure.Code())
}

func TestMissingEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {}))
	defer server.Close()

	enricher, failure := buildContentTargeting(server.Client(), "")

	assert.Empty(t, failure)

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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Equal(t, expectedRequest, request)
	assert.Equal(t, TargetingUrlErrorCode, targetingFailure.Code())
}

func TestRequestAlreadyHasSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	enricher, _ := buildContentTargeting(server.Client(), server.URL)

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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Data: []openrtb2.Data{
					{
						Name: "jwplayer.com",
						Segment: []openrtb2.Segment{
							{Value: "1"}, {Value: "2"}, {Value: "3"},
						},
						Ext: []byte(`{"segtax": 502}`),
					},
				},
				Ext: json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Data: []openrtb2.Data{
					{
						Name: "jwplayer.com",
						Segment: []openrtb2.Segment{
							{Value: "1"}, {Value: "2"}, {Value: "3"},
						},
						Ext: []byte(`{"segtax": 502}`),
					},
				},
				Ext: json.RawMessage(`{"description"": "testDesc"`),
			},
			Keywords: "jwpseg=1,jwpseg=2,jwpseg=3",
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Equal(t, expectedRequest, request)
	assert.Empty(t, targetingFailure)

	request.Site.Keywords = "existingKey=value"

	expectedRequest = &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Data: []openrtb2.Data{
					{
						Name: "jwplayer.com",
						Segment: []openrtb2.Segment{
							{Value: "1"}, {Value: "2"}, {Value: "3"},
						},
						Ext: []byte(`{"segtax": 502}`),
					},
				},
				Ext: json.RawMessage(`{"description"": "testDesc"`),
			},
			Keywords: "existingKey=value,jwpseg=1,jwpseg=2,jwpseg=3",
		},
	}
	targetingFailure = enricher.EnrichRequest(request, "testSiteId")
	assert.Empty(t, targetingFailure)
	assert.Equal(t, expectedRequest, request)
}

func TestMissingSiteId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	enricher, _ := buildContentTargeting(server.Client(), server.URL)

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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "")
	assert.Equal(t, expectedRequest, request)
	assert.Equal(t, MissingSiteIdErrorCode, targetingFailure.Code())
}

func TestMissingTemplate(t *testing.T) {
	enricher := ContentTargeting{}
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "")
	assert.Equal(t, expectedRequest, request)
	assert.Equal(t, EmptyTemplateErrorCode, targetingFailure.Code())
}

func TestMissingContentUrl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	enricher, _ := buildContentTargeting(server.Client(), server.URL)

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
			Content: &openrtb2.Content{
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Equal(t, expectedRequest, request)
	assert.Equal(t, MissingMediaUrlErrorCode, targetingFailure.Code())
}

func Test404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(404)
	}))
	defer server.Close()
	enricher, _ := buildContentTargeting(server.Client(), server.URL)

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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Equal(t, expectedRequest, request)
	assert.Equal(t, BaseNetworkErrorCode+404, targetingFailure.Code())
}

func TestEmptySegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(200)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": [], "targeting_profiles": []}}`))
	}))
	defer server.Close()
	enricher, _ := buildContentTargeting(server.Client(), server.URL)

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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	expectedRequest := &openrtb2.BidRequest{
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
			Content: &openrtb2.Content{
				URL:   "http://www.testUrl.com/media.mp4",
				Title: "testTitle",
				Ext:   json.RawMessage(`{"description"": "testDesc"`),
			},
		},
	}

	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Equal(t, expectedRequest, request)
	assert.Equal(t, EmptyTargetingSegmentsErrorCode, targetingFailure.Code())
}
