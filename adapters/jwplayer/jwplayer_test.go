package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/stretchr/testify/assert"
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
			ID: "test_imp_id",
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

	assert.Len(t, resultJSON.Imp, 1, "Imp count should be equal or less than Imps from input. In this test, should be 1.")
	assert.Empty(t, resultJSON.Imp[0].Ext, "Ext should be deleted")
	assert.Equal(t, "test_placement_id" , resultJSON.Imp[0].TagID, "placement id should be set to TagID")
	assert.Equal(t, int(resultJSON.Imp[0].Video.H), 250, "extra metadata in Imp should not be removed")
	assert.Equal(t, int(resultJSON.Imp[0].Video.W), 350, "extra metadata in Imp should not be removed")
}

func TestInvalidImpExt(t *testing.T) {
	var a JWPlayerAdapter
	a.endpoint = "http://test.com/openrtb2"

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	request := &openrtb2.BidRequest{
		ID: "test_id_1",
		Imp: []openrtb2.Imp{{
			ID: "test_imp_id",
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
			ID: "test_imp_id",
			Ext: json.RawMessage(`{"bidder":{"placementId": "test_placement_id"}}`),
		}},
		Site: &openrtb2.Site{
			ID: "test_site_id",
			Domain: "test_domain",
			Publisher: &openrtb2.Publisher{
				ID: "test_publisher_id",
				Name: "testPublisher_name",
			},
		},
		App: &openrtb2.App{
			ID: "test_app_id",
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
