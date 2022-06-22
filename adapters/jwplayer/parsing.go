package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"net/url"
	"strings"
)

func ParseContentMetadata(content openrtb2.Content) jwContentMetadata {
	metadata := jwContentMetadata{
		Url:   content.URL,
		Title: content.Title,
	}

	contentExt := jwContentExt{}
	if error := json.Unmarshal(content.Ext, &contentExt); error == nil {
		println("karim: DESCRIPTION: ", contentExt.Description)
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
	dataExt := jwDataExt{}
	if error := json.Unmarshal(datum.Ext, &dataExt); error != nil {
		return false
	}

	return datum.Name == jwplayerDomain && dataExt.Segtax == jwplayerSegtax && len(datum.Segment) > 0
}

func isValidMediaUrl(rawUrl string) bool {
	parsedUrl, error := url.ParseRequestURI(rawUrl)
	if error != nil {
		return false
	}
	isLocalFile := parsedUrl.Scheme == "file"
	isLocalHost := parsedUrl.Opaque != ""
	isRelativePath := parsedUrl.Host == ""
	return !isLocalFile && !isRelativePath && !isLocalHost
}

func ParseJwpsegs(segments []openrtb2.Segment) []string {
	jwpsegs := make([]string, len(segments))
	for index, segment := range segments {
		jwpsegs[index] = segment.Value
	}

	return jwpsegs
}

func GetAllJwpsegs(targeting jwTargetingData) []string {
	return append(targeting.BaseSegments, targeting.TargetingProfiles...)
}

func MakeOrtbDatum(jwpsegs []string) (contentData openrtb2.Data) {
	contentData.Name = jwplayerDomain
	contentData.Segment = MakeOrtbSegments(jwpsegs)
	dataExt := jwDataExt{
		Segtax: jwplayerSegtax,
	}
	contentData.Ext, _ = json.Marshal(dataExt)
	return contentData
}

func MakeOrtbSegments(jwpsegs []string) []openrtb2.Segment {
	segments := make([]openrtb2.Segment, len(jwpsegs))
	for index, jwpseg := range jwpsegs {
		segment := openrtb2.Segment{
			Value: jwpseg,
		}
		segments[index] = segment
	}

	return segments
}

func writeToKeywords(keywords *string, jwpsegs []string) {
	if len(jwpsegs) == 0 {
		return
	}

	if len(*keywords) > 0 {
		*keywords += ","
	}

	*keywords += GetXandrKeywords(jwpsegs)
}

func GetXandrKeywords(jwpsegs []string) string {
	if len(jwpsegs) == 0 {
		return ""
	}

	keyword := "jwpseg="
	// expected format: jwpseg=1,jwpseg=2,jwpseg=3
	keyword += strings.Join(jwpsegs, ","+keyword)
	return keyword
}
