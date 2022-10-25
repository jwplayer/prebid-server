package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseBidderParams(t *testing.T) {
	params, err := ParseBidderParams(openrtb2.Imp{
		Ext: json.RawMessage(`{"bidder":{"placementId": "1"}}`),
	})
	assert.Empty(t, err)
	assert.Equal(t, "1", params.PlacementId)

	params, err = ParseBidderParams(openrtb2.Imp{
		Ext: json.RawMessage(`{"else":{"placementId": "1"}}`),
	})
	assert.NotNil(t, err)
	assert.Empty(t, params)

	params, err = ParseBidderParams(openrtb2.Imp{
		Ext: json.RawMessage(`{"bidder":{"otherId": "1"}}`),
	})
	assert.NotNil(t, err)
	assert.Empty(t, params)
}

func TestContentMetadataParseSuccess(t *testing.T) {
	description := "Test Description"
	descriptionExt, _ := json.Marshal(ContentExt{
		Description: description,
	})
	content := openrtb2.Content{
		URL:   "//test.medial.url/file.mp4",
		Title: "Test title",
		Ext:   descriptionExt,
	}

	metadata := ParseContentMetadata(content)

	expectedMetadata := ContentMetadata{
		Url:         "//test.medial.url/file.mp4",
		Title:       "Test title",
		Description: "Test Description",
	}

	assert.Equal(t, expectedMetadata, metadata)
}

func TestGetExistingJwpsegs(t *testing.T) {
	externalSegments1 := []openrtb2.Segment{{ID: "sthg"}, {ID: "else"}}
	externalData1 := openrtb2.Data{Name: "external", Segment: externalSegments1}

	externalSegments2 := []openrtb2.Segment{{ID: "sthg2"}, {ID: "else2"}}
	externalData2 := openrtb2.Data{Name: "external number 2", Segment: externalSegments2}

	jwSegments := []openrtb2.Segment{{ID: "1"}, {ID: "2"}}
	dataExt := DataExt{Segtax: 502}
	ext, _ := json.Marshal(dataExt)
	jwData := openrtb2.Data{Name: "jwplayer.com", Segment: jwSegments, Ext: ext}

	externalSegments3 := []openrtb2.Segment{{ID: "3"}, {ID: "4"}}
	dataWithoutSegtax := openrtb2.Data{Name: "jwplayer.com", Segment: externalSegments3}

	externalSegments4 := []openrtb2.Segment{{ID: "5"}, {ID: "6"}}
	dataWithWrongName := openrtb2.Data{Name: "other.com", Segment: externalSegments4, Ext: ext}

	dataWithEmptySegments := openrtb2.Data{Name: "jwplayer.com", Ext: ext}

	jwpsegs := GetExistingJwpsegs([]openrtb2.Data{externalData1, dataWithoutSegtax, dataWithWrongName, dataWithEmptySegments, jwData, externalData2})
	expectedJwpsegs := []string{"1", "2"}

	assert.Len(t, jwpsegs, len(expectedJwpsegs))
	assert.ElementsMatch(t, jwpsegs, expectedJwpsegs)
}

func TestHasJwpsegs(t *testing.T) {
	segments := []openrtb2.Segment{{
		ID: "88888888",
	}}
	jwDatumExt, _ := json.Marshal(DataExt{Segtax: jwplayerSegtax})
	datum := openrtb2.Data{
		Name:    "jwplayer.com",
		Segment: segments,
		Ext:     jwDatumExt,
	}

	assert.True(t, HasJwpsegs(datum))

	datum.Name = "other"
	assert.False(t, HasJwpsegs(datum))

	datum.Name = "jwplayer.com"
	datum.Ext, _ = json.Marshal(ContentExt{Description: "descr"})
	assert.False(t, HasJwpsegs(datum))

	datum.Ext = jwDatumExt
	datum.Segment = []openrtb2.Segment{}
	assert.False(t, HasJwpsegs(datum))
}

func TestParseJwpsegs(t *testing.T) {
	var emptySegments []openrtb2.Segment
	emptyJwpsegs := ParseJwpsegs(emptySegments)
	assert.Empty(t, emptyJwpsegs)

	segments := []openrtb2.Segment{{ID: "1"}, {ID: "2"}, {ID: "3"}}
	jwpsegs := ParseJwpsegs(segments)
	expectedJwpsegs := []string{"1", "2", "3"}
	assert.Len(t, jwpsegs, len(segments))
	assert.ElementsMatch(t, expectedJwpsegs, jwpsegs)
}

func TestMakeOrtbDatum(t *testing.T) {
	jwpsegs := []string{"1", "2", "3"}
	datum := MakeOrtbDatum(jwpsegs)

	expectedDatum := openrtb2.Data{
		Name:    "jwplayer.com",
		Segment: []openrtb2.Segment{{ID: "1"}, {ID: "2"}, {ID: "3"}},
		Ext:     json.RawMessage(`{"segtax":502}`),
	}

	assert.Equal(t, expectedDatum, datum)
}

func TestMakeOrtbSegments(t *testing.T) {
	var emptyJwpsegs []string
	assert.Empty(t, MakeOrtbSegments(emptyJwpsegs))

	jwpsegs := []string{"1", "2", "3"}
	segments := MakeOrtbSegments(jwpsegs)
	expectedSegments := []openrtb2.Segment{{ID: "1"}, {ID: "2"}, {ID: "3"}}
	assert.Len(t, segments, len(expectedSegments))
	assert.ElementsMatch(t, segments, expectedSegments)
}
