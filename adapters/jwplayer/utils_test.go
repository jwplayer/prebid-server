package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseExtraInfo(t *testing.T) {
	extraInfo := parseExtraInfo("{\"targeting_endpoint\": \"targetingUrl\"}")
	assert.Equal(t, "targetingUrl", extraInfo.TargetingEndpoint)

	defaultTargetingUrl := "https://content-targeting-api.longtailvideo.com/property/{{.SiteId}}/content_segments?content_url=%{{.MediaUrl}}&title={{.Title}}&description={{.Description}}"
	extraInfo = parseExtraInfo("{/")
	assert.Equal(t, defaultTargetingUrl, extraInfo.TargetingEndpoint)

	extraInfo = parseExtraInfo("{}")
	assert.Equal(t, defaultTargetingUrl, extraInfo.TargetingEndpoint)
}

//GetAppnexusExt
//ParsePublisherParams
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
	assert.Equal(t, content.URL, metadata.Url)
	assert.Equal(t, content.Title, metadata.Title)
	assert.Equal(t, description, metadata.Description)
}

func TestGetExistingJwpsegs(t *testing.T) {
	externalSegments1 := []openrtb2.Segment{{Value: "sthg"}, {Value: "else"}}
	externalData1 := openrtb2.Data{Name: "external", Segment: externalSegments1}

	externalSegments2 := []openrtb2.Segment{{Value: "sthg2"}, {Value: "else2"}}
	externalData2 := openrtb2.Data{Name: "external number 2", Segment: externalSegments2}

	jwSegments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}}
	dataExt := DataExt{Segtax: 502}
	ext, _ := json.Marshal(dataExt)
	jwData := openrtb2.Data{Name: "jwplayer.com", Segment: jwSegments, Ext: ext}

	externalSegments3 := []openrtb2.Segment{{Value: "3"}, {Value: "4"}}
	dataWithoutSegtax := openrtb2.Data{Name: "jwplayer.com", Segment: externalSegments3}

	externalSegments4 := []openrtb2.Segment{{Value: "5"}, {Value: "6"}}
	dataWithWrongName := openrtb2.Data{Name: "other.com", Segment: externalSegments4, Ext: ext}

	dataWithEmptySegments := openrtb2.Data{Name: "jwplayer.com", Ext: ext}

	jwpsegs := GetExistingJwpsegs([]openrtb2.Data{externalData1, dataWithoutSegtax, dataWithWrongName, dataWithEmptySegments, jwData, externalData2})
	expectedJwpsegs := []string{"1", "2"}

	assert.Len(t, jwpsegs, len(expectedJwpsegs))
	assert.ElementsMatch(t, jwpsegs, expectedJwpsegs)
}

func TestHasJwpsegs(t *testing.T) {
	segments := []openrtb2.Segment{{
		Value: "88888888",
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

func TestIsValidMediaUrl(t *testing.T) {
	assert.False(t, IsValidMediaUrl(""))
	assert.False(t, IsValidMediaUrl("nothing"))
	assert.False(t, IsValidMediaUrl("media.mp4"))
	assert.False(t, IsValidMediaUrl("file://hello.com/media.mp4"))
	assert.False(t, IsValidMediaUrl("localhost:9999/hello.com/media.mp4"))
	assert.False(t, IsValidMediaUrl("0.0.0.0:9999/hello.com/media.mp4"))
	assert.False(t, IsValidMediaUrl("/hello.com/media.mp4"))

	assert.True(t, IsValidMediaUrl("//hello.com/media.mp4"))
	assert.True(t, IsValidMediaUrl("http://hello.com/media.mp4"))
	assert.True(t, IsValidMediaUrl("https://hello.com/media.mp4"))
	assert.True(t, IsValidMediaUrl("https://hello.com/media.mp4?additional=sthg"))
}

//BuildTargetingEndpoint

func TestParseJwpsegs(t *testing.T) {
	var emptySegments []openrtb2.Segment
	emptyJwpsegs := ParseJwpsegs(emptySegments)
	assert.Empty(t, emptyJwpsegs)

	segments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "3"}}
	jwpsegs := ParseJwpsegs(segments)
	expectedJwpsegs := []string{"1", "2", "3"}
	assert.Len(t, jwpsegs, len(segments))
	assert.ElementsMatch(t, expectedJwpsegs, jwpsegs)
}

func TestGetAllJwpsegs(t *testing.T) {
	targetingData := TargetingData{
		BaseSegments:      []string{"1", "2", "3"},
		TargetingProfiles: []string{"4", "5"},
	}

	jwpsegs := GetAllJwpsegs(targetingData)
	expectedJwpsegs := []string{"1", "2", "3", "4", "5"}

	assert.Len(t, jwpsegs, len(expectedJwpsegs))
	assert.ElementsMatch(t, jwpsegs, expectedJwpsegs)
}

func TestMakeOrtbDatum(t *testing.T) {
	jwpsegs := []string{"1", "2", "3"}
	datum := MakeOrtbDatum(jwpsegs)

	assert.Equal(t, datum.Name, "jwplayer.com")

	var dataExt DataExt
	json.Unmarshal(datum.Ext, &dataExt)
	assert.Equal(t, dataExt.Segtax, 502)

	expectedSegments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "3"}}
	assert.Len(t, datum.Segment, len(expectedSegments))
	assert.ElementsMatch(t, datum.Segment, expectedSegments)
}

func TestMakeOrtbSegments(t *testing.T) {
	var emptyJwpsegs []string
	assert.Empty(t, MakeOrtbSegments(emptyJwpsegs))

	jwpsegs := []string{"1", "2", "3"}
	segments := MakeOrtbSegments(jwpsegs)
	expectedSegments := []openrtb2.Segment{{Value: "1"}, {Value: "2"}, {Value: "3"}}
	assert.Len(t, segments, len(expectedSegments))
	assert.ElementsMatch(t, segments, expectedSegments)
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
