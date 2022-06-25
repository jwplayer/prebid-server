package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchContentTargetingSuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(http.StatusOK)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2", "3", "4"], "targeting_profiles": ["5", "6", "7", "8"]}}`))
	}))
	defer server.Close()

	enricher, failure := buildRequestEnricher(server.Client(), server.URL)

	assert.Empty(t, failure)

	metadata := jwContentMetadata{
		Url:         "http://www.testUrl.com/media.mp4",
		Title:       "testTitle",
		Description: "testDesc",
	}

	response, fetchError := enricher.FetchContentTargeting("testSiteId", metadata)
	assert.Empty(t, fetchError)
	assert.Equal(t, response.Uuid, "test_uuid")
	assert.Equal(t, response.Data.MediaId, "test_id")

	assert.Len(t, response.Data.TargetingProfiles, 4)
	expectedTpis := []string{"5", "6", "7", "8"}
	assert.ElementsMatch(t, response.Data.TargetingProfiles, expectedTpis)

	assert.Len(t, response.Data.BaseSegments, 4)
	expectedBaseSegs := []string{"1", "2", "3", "4"}
	assert.ElementsMatch(t, response.Data.BaseSegments, expectedBaseSegs)
}

func TestFetchContentTargetingDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(http.StatusOK)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2", `))
	}))
	defer server.Close()

	enricher, failure := buildRequestEnricher(server.Client(), server.URL)

	assert.Empty(t, failure)

	metadata := jwContentMetadata{
		Url:         "http://www.testUrl.com/media.mp4",
		Title:       "testTitle",
		Description: "testDesc",
	}

	response, fetchError := enricher.FetchContentTargeting("testSiteId", metadata)
	assert.Empty(t, response)
	assert.Equal(t, BaseDecodingErrorCode, fetchError.Code())
}

func TestFetchContentTargetingNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(433)
	}))
	defer server.Close()

	enricher, failure := buildRequestEnricher(server.Client(), server.URL)

	assert.Empty(t, failure)

	metadata := jwContentMetadata{
		Url:         "http://www.testUrl.com/media.mp4",
		Title:       "testTitle",
		Description: "testDesc",
	}

	response, fetchError := enricher.FetchContentTargeting("testSiteId", metadata)
	assert.Empty(t, response)
	assert.Equal(t, BaseNetworkErrorCode+433, fetchError.Code())
}

func TestFetchContentTargetingBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {}))
	defer server.Close()

	enricher, failure := buildRequestEnricher(server.Client(), "")

	assert.Empty(t, failure)

	metadata := jwContentMetadata{
		Url:         "http://www.testUrl.com/media.mp4",
		Title:       "testTitle",
		Description: "testDesc",
	}

	response, fetchError := enricher.FetchContentTargeting("testSiteId", metadata)
	assert.Empty(t, response)
	assert.Equal(t, HttpRequestExecutionErrorCode, fetchError.Code())
}

func TestRequestHasSegments(t *testing.T) {
	keywords := ""
	content := openrtb2.Content{
		Data: []openrtb2.Data{
			{
				Name: "jwplayer.com",
				Segment: []openrtb2.Segment{
					{Value: "1"}, {Value: "2"}, {Value: "3"},
				},
				Ext: []byte(`{"segtax": 502}`),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	enricher, _ := buildRequestEnricher(server.Client(), server.URL)

	enricher.enrich(&keywords, &content, "testId")
	assert.Equal(t, "jwpseg=1,jwpseg=2,jwpseg=3", keywords)

	keywords = "existingKey=value"
	enricher.enrich(&keywords, &content, "testId")
	assert.Equal(t, "existingKey=value,jwpseg=1,jwpseg=2,jwpseg=3", keywords)
}

func TestEnrichMissingSiteId(t *testing.T) {
	keywords := ""
	content := openrtb2.Content{}
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	enricher, _ := buildRequestEnricher(server.Client(), server.URL)
	failure := enricher.enrich(&keywords, &content, "")
	assert.Equal(t, MissingSiteIdErrorCode, failure.Code())
}

func TestEnrichMissingContentUrl(t *testing.T) {
	keywords := ""
	content := openrtb2.Content{}
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	enricher, _ := buildRequestEnricher(server.Client(), server.URL)
	failure := enricher.enrich(&keywords, &content, "testId")
	assert.Equal(t, MissingMediaUrlErrorCode, failure.Code())
}

func TestEnrichFetchError(t *testing.T) {
	keywords := ""
	content := openrtb2.Content{URL: "http://test.com/media.mp4"}
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(404)
	}))
	defer server.Close()
	enricher, _ := buildRequestEnricher(server.Client(), server.URL)
	failure := enricher.enrich(&keywords, &content, "testId")
	assert.Equal(t, BaseNetworkErrorCode+404, failure.Code())
}

func TestEnrichErrorEmptySegments(t *testing.T) {
	keywords := ""
	content := openrtb2.Content{URL: "http://test.com/media.mp4"}
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(200)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": [], "targeting_profiles": []}}`))
	}))
	defer server.Close()
	enricher, _ := buildRequestEnricher(server.Client(), server.URL)
	failure := enricher.enrich(&keywords, &content, "testId")
	assert.Equal(t, EmptyTargetingSegmentsErrorCode, failure.Code())
}

func TestEnrichSuccess(t *testing.T) {
	keywords := "existingKey=value"
	content := openrtb2.Content{URL: "http://test.com/media.mp4"}
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(200)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2"], "targeting_profiles": ["5", "6"]}}`))
	}))
	defer server.Close()
	enricher, _ := buildRequestEnricher(server.Client(), server.URL)
	failure := enricher.enrich(&keywords, &content, "testId")
	assert.Empty(t, failure)
	assert.Equal(t, "existingKey=value,jwpseg=1,jwpseg=2,jwpseg=5,jwpseg=6", keywords)
	datum := content.Data[0]
	assert.Equal(t, "jwplayer.com", datum.Name)
	assert.Len(t, datum.Segment, 4)
	expectedSegments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "5"}, {Value: "6"}}
	assert.ElementsMatch(t, datum.Segment, expectedSegments)

	expectedExt := jwDataExt{Segtax: 502}
	datumExt := jwDataExt{}
	json.Unmarshal(datum.Ext, &datumExt)
	assert.Equal(t, expectedExt, datumExt)
}

func TestEnrichSuccessAppendsToPreviousData(t *testing.T) {
	keywords := ""
	content := openrtb2.Content{
		URL: "http://test.com/media.mp4",
		Data: []openrtb2.Data{{
			Name: "otherData",
			ID:   "otherDataId",
		}, {
			Name: "3rdData",
			ID:   "3rdDataId",
		}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
		respWriter.WriteHeader(200)
		respWriter.Write([]byte(`{"uuid": "test_uuid", "data": {"media_id": "test_id", "base_segments": ["1", "2"], "targeting_profiles": ["5", "6"]}}`))
	}))
	defer server.Close()
	enricher, _ := buildRequestEnricher(server.Client(), server.URL)
	failure := enricher.enrich(&keywords, &content, "testId")
	assert.Empty(t, failure)
	assert.Equal(t, "jwpseg=1,jwpseg=2,jwpseg=5,jwpseg=6", keywords)
	assert.Len(t, content.Data, 3)
	otherData := content.Data[0]
	assert.Equal(t, otherData.Name, "otherData")
	assert.Equal(t, otherData.ID, "otherDataId")
	thirdData := content.Data[1]
	assert.Equal(t, thirdData.Name, "3rdData")
	assert.Equal(t, thirdData.ID, "3rdDataId")
	datum := content.Data[2]
	assert.Equal(t, "jwplayer.com", datum.Name)
	assert.Len(t, datum.Segment, 4)
	expectedSegments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "5"}, {Value: "6"}}
	assert.ElementsMatch(t, datum.Segment, expectedSegments)

	expectedExt := jwDataExt{Segtax: 502}
	datumExt := jwDataExt{}
	json.Unmarshal(datum.Ext, &datumExt)
	assert.Equal(t, expectedExt, datumExt)
}
