package jwplayer

import (
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRequest(t *testing.T) {
	var a JWPlayerAdapter
	a.endpoint = "http://test.com/openrtb2"

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	var req openrtb2.BidRequest
	req.ID = "test_id"

	impExt := `{"prebid":{"bidder":{"jwplayer":{"placementId":"123"}}}}`

	req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_0", Ext: []byte(impExt)})

	result, err := a.MakeRequests(&req, &reqInfo)

	assert.Empty(t, err, "Errors array should be empty")
	assert.Len(t, result, 1, "Only one request should be returned")
}


func TestInvalidImpExt(t *testing.T) {
	var a JWPlayerAdapter
	a.endpoint = "http://test.com/openrtb2"

	var reqInfo adapters.ExtraRequestInfo
	reqInfo.PbsEntryPoint = "video"

	var req1 openrtb2.BidRequest
	req1.ID = "test_id_1"

	impExt1 := `{}`

	req1.Imp = append(req1.Imp, openrtb2.Imp{ID: "1_0", Ext: []byte(impExt1)})

	result, err := a.MakeRequests(&req1, &reqInfo)

	assert.Len(t, err, 1, "An error should be returned")
	assert.Empty(t, result, "Result should be nil")

		var req2 openrtb2.BidRequest
		req2.ID = "test_id_2"

		impExt2 := `{"bidder":{"jwplayer":{"placementId": "123"}}}`

		req2.Imp = append(req2.Imp, openrtb2.Imp{ID: "2_0", Ext: []byte(impExt2)})

		result2, err2 := a.MakeRequests(&req2, &reqInfo)

		assert.Len(t, err2, 1, "An error should be returned")
		assert.Empty(t, result2, "Result should be nil")

}
