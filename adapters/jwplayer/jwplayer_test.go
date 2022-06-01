package jwplayer

import (
    "testing"
    "JWPlayerAdapter"
    "github.com/prebid/prebid-server/adapters"
    "github.com/mxmCherry/openrtb/v15/openrtb2"
    "github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {
    var a JWPlayerAdapter
    a.URI = "http://test.com/openrtb2"

    var reqInfo adapters.ExtraRequestInfo
    reqInfo.PbsEntryPoint = "video"

    var req openrtb2.BidRequest
    req.ID = "test_id"

    impExt := `{"prebid":{"bidder":{"jwplayer":{"placementId":123}}}}`

    req.Imp = append(req.Imp, openrtb2.Imp{ID: "1_0", Ext: []byte(impExt)})

    result, err := a.MakeRequests(&req, &reqInfo)

    assert.Empty(t, err, "Errors array should be empty")
    assert.Len(t, result, 1, "Only one request should be returned")

//     var error error
//     var reqData *openrtb2.BidRequest
//     error = json.Unmarshal(result[0].Body, &reqData)
//     assert.NoError(t, error, "Response body unmarshalling error should be nil")
//
//     var reqDataExt *appnexusReqExt
//     error = json.Unmarshal(reqData.Ext, &reqDataExt)
//     assert.NoError(t, error, "Response ext unmarshalling error should be nil")
//
//     regMatch, matchErr := regexp.Match(`^[0-9]+$`, []byte(reqDataExt.Appnexus.AdPodId))
//     assert.NoError(t, matchErr, "Regex match error should be nil")
//     assert.True(t, regMatch, "AdPod id doesn't present in Appnexus extension or has incorrect format")
}
