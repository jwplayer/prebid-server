package jwplayer

import (
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
	assert.Equal(t, 305000, fetchError.Code())
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
	assert.Equal(t, 304433, fetchError.Code())
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
	assert.Equal(t, 303050, fetchError.Code())
}

func TestRequestHasSegments(t *testing.T) {

}

func TestMissingPublisherId(t *testing.T) {

}

func TestInvalidContentUrl(t *testing.T) {

}
