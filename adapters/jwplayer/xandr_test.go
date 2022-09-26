package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetXandrImpExt(t *testing.T) {
	appnexusExt := GetXandrImpExt("1234")
	expectedJSON := json.RawMessage(`{"appnexus":{"placement_id":1234}}`)
	assert.Equal(t, expectedJSON, appnexusExt)
}

func TestSetXandrVideoExt(t *testing.T) {
	video := &openrtb2.Video{}
	SetXandrVideoExt(video)
	assert.Empty(t, video.Ext)

	video.Placement = openrtb2.VideoPlacementTypeInArticle
	SetXandrVideoExt(video)
	expectedVideoExt := json.RawMessage(`{"appnexus":{"context":4}}`)
	assert.Equal(t, expectedVideoExt, video.Ext)

	video.Ext = nil
	video.Placement = openrtb2.VideoPlacementTypeInStream
	video.StartDelay = openrtb2.StartDelayGenericPostRoll.Ptr()
	SetXandrVideoExt(video)
	expectedVideoExt = json.RawMessage(`{"appnexus":{"context":3}}`)
	assert.Equal(t, expectedVideoExt, video.Ext)
}

func TestGetXandrContext(t *testing.T) {
	video := openrtb2.Video{}
	assert.Equal(t, Unknown, GetXandrContext(video))

	video.Placement = openrtb2.VideoPlacementTypeInBanner
	video.StartDelay = openrtb2.StartDelayPreRoll.Ptr()
	assert.Equal(t, Outstream, GetXandrContext(video))

	video.Placement = openrtb2.VideoPlacementTypeInStream
	assert.Equal(t, PreRoll, GetXandrContext(video))
}

func TestGetXandrContextFromStartdelay(t *testing.T) {
	assert.Equal(t, PreRoll, GetXandrContextFromStartdelay(openrtb2.StartDelayPreRoll))
	assert.Equal(t, MidRoll, GetXandrContextFromStartdelay(openrtb2.StartDelayGenericMidRoll))
	assert.Equal(t, MidRoll, GetXandrContextFromStartdelay(openrtb2.StartDelay(5)))
	assert.Equal(t, PostRoll, GetXandrContextFromStartdelay(openrtb2.StartDelayGenericPostRoll))
}

func TestConvertToXandrKeywords(t *testing.T) {
	var emptyJwpsegs []string
	assert.Equal(t, "", ConvertToXandrKeywords(emptyJwpsegs))

	singleJwpseg := []string{"80808080"}
	assert.Equal(t, "jwpseg=80808080", ConvertToXandrKeywords(singleJwpseg))

	multipleJwpsegs := []string{"88888888", "80808080", "80088008"}
	assert.Equal(t, "jwpseg=88888888,jwpseg=80808080,jwpseg=80088008", ConvertToXandrKeywords(multipleJwpsegs))
}

func TestWriteToXandrKeywords(t *testing.T) {
	keyword := "key=value"
	var jwpsegs []string
	WriteToXandrKeywords(&keyword, jwpsegs)
	assert.Equal(t, "key=value", keyword)

	jwpsegs = append(jwpsegs, "80808080")
	WriteToXandrKeywords(&keyword, jwpsegs)
	assert.Equal(t, "key=value,jwpseg=80808080", keyword)

}
