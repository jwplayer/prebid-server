package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"strings"
)

func ParseContentMetadata(content openrtb2.Content) JWContentMetadata {
	metadata := JWContentMetadata{
		Url: content.URL,
		Title: content.Title,
	}

	contentExt := JWContentExt{}
	if error := json.Unmarshal(content.Ext, &contentExt); error == nil {
		metadata.Description = contentExt.Description
	}

	return metadata
}

func GetExistingJwpsegs(data []openrtb2.Data) []string {
	for _, datum := range data {
		if HasJwpsegs(datum) {
			return ParseJwpsegs(datum.Segment)
		}
	}

	return nil
}

func HasJwpsegs(datum openrtb2.Data) bool {
	dataExt := JWDataExt{}
	if error := json.Unmarshal(datum.Ext, &dataExt); error != nil {
		return false
	}

	return datum.Name == jwplayerDomain && dataExt.Segtax == jwplayerSegtax
}

func ParseJwpsegs(segments []openrtb2.Segment) []string {
	jwpsegs := make([]string, len(segments))
	for _, segment := range segments {
		jwpsegs = append(jwpsegs, segment.Value)
	}

	return jwpsegs
}

func GetAllJwpsegs(targeting JWTargetingData) []string {
	return append(targeting.BaseSegments, targeting.TargetingProfiles...)
}

func MakeOrtbDatum(jwpsegs []string) (contentData openrtb2.Data) {
	contentData.Name = jwplayerDomain
	contentData.Segment = MakeOrtbSegments(jwpsegs)
	dataExt := JWDataExt{
		Segtax: jwplayerSegtax,
	}
	contentData.Ext, _ = json.Marshal(dataExt)
	return contentData
}

func MakeOrtbSegments(jwpsegs []string) []openrtb2.Segment {
	segments := make([]openrtb2.Segment, len(jwpsegs))
	for _, jwpseg := range jwpsegs {
		segment := openrtb2.Segment{
			Value: jwpseg,
		}
		segments = append(segments, segment)
	}

	return segments
}

func GetXandrKeywords(jwpsegs []string) string {
	if len(jwpsegs) == 0 {
		return ""
	}

	keyword := "jwpseg="
	// expected format: jwpseg=1,jwpseg=2,jwpseg=3
	keyword += strings.Join(jwpsegs, "," + keyword)
	return keyword
}
