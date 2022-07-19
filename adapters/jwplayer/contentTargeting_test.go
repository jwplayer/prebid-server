package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
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
	assert.Equal(t, "jwpseg=1,jwpseg=2,jwpseg=3,jwpseg=4,jwpseg=5,jwpseg=6,jwpseg=7,jwpseg=8", request.App.Keywords)
	datum := request.App.Content.Data[0]
	assert.Equal(t, "jwplayer.com", datum.Name)
	assert.Len(t, datum.Segment, 8)
	expectedSegments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "3"}, {Value: "4"}, {Value: "5"}, {Value: "6"}, {Value: "7"}, {Value: "8"}}
	assert.ElementsMatch(t, datum.Segment, expectedSegments)

	expectedExt := DataExt{Segtax: 502}
	datumExt := DataExt{}
	json.Unmarshal(datum.Ext, &datumExt)
	assert.Equal(t, expectedExt, datumExt)
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
	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Empty(t, targetingFailure)
	assert.Equal(t, "existingKey=value,jwpseg=1,jwpseg=2,jwpseg=5,jwpseg=6", request.Site.Keywords)
	datum := request.Site.Content.Data[0]
	assert.Equal(t, "jwplayer.com", datum.Name)
	assert.Len(t, datum.Segment, 4)
	expectedSegments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "5"}, {Value: "6"}}
	assert.ElementsMatch(t, datum.Segment, expectedSegments)

	expectedExt := DataExt{Segtax: 502}
	datumExt := DataExt{}
	json.Unmarshal(datum.Ext, &datumExt)
	assert.Equal(t, expectedExt, datumExt)
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
	targetingFailure := enricher.EnrichRequest(request, "testId")

	assert.Empty(t, targetingFailure)
	assert.Equal(t, "jwpseg=1,jwpseg=2,jwpseg=5,jwpseg=6", request.Site.Keywords)
	assert.Len(t, request.Site.Content.Data, 3)
	otherData := request.Site.Content.Data[0]
	assert.Equal(t, otherData.Name, "otherData")
	assert.Equal(t, otherData.ID, "otherDataId")
	thirdData := request.Site.Content.Data[1]
	assert.Equal(t, thirdData.Name, "3rdData")
	assert.Equal(t, thirdData.ID, "3rdDataId")
	datum := request.Site.Content.Data[2]
	assert.Equal(t, "jwplayer.com", datum.Name)
	assert.Len(t, datum.Segment, 4)
	expectedSegments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "5"}, {Value: "6"}}
	assert.ElementsMatch(t, datum.Segment, expectedSegments)

	expectedExt := DataExt{Segtax: 502}
	datumExt := DataExt{}
	json.Unmarshal(datum.Ext, &datumExt)
	assert.Equal(t, expectedExt, datumExt)
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

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Empty(t, request.App.Keywords)
	assert.Empty(t, request.App.Content)
	assert.Equal(t, MissingContentBlockErrorCode, targetingFailure.Code())
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

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Empty(t, request.App.Keywords)
	assert.Empty(t, request.App.Content.Data)
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

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Empty(t, request.Site.Keywords)
	assert.Empty(t, request.Site.Content.Data)
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

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Empty(t, request.Site.Keywords)
	assert.Empty(t, request.Site.Content.Data)
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

	targetingFailure := enricher.EnrichRequest(request, "testSiteId")
	assert.Empty(t, targetingFailure)
	assert.Equal(t, "jwpseg=1,jwpseg=2,jwpseg=3", request.Site.Keywords)

	request.Site.Keywords = "existingKey=value"
	targetingFailure = enricher.EnrichRequest(request, "testSiteId")
	assert.Empty(t, targetingFailure)
	assert.Equal(t, "existingKey=value,jwpseg=1,jwpseg=2,jwpseg=3", request.Site.Keywords)
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

	targetingFailure := enricher.EnrichRequest(request, "")
	assert.Empty(t, request.Site.Keywords)
	assert.Empty(t, request.Site.Content.Data)
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

	targetingFailure := enricher.EnrichRequest(request, "")
	assert.Empty(t, request.Site.Keywords)
	assert.Empty(t, request.Site.Content.Data)
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

	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Empty(t, request.Site.Keywords)
	assert.Empty(t, request.Site.Content.Data)
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

	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Empty(t, request.Site.Keywords)
	assert.Empty(t, request.Site.Content.Data)
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

	targetingFailure := enricher.EnrichRequest(request, "testId")
	assert.Empty(t, request.Site.Keywords)
	assert.Empty(t, request.Site.Content.Data)
	assert.Equal(t, EmptyTargetingSegmentsErrorCode, targetingFailure.Code())
}
