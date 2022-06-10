package utils

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"net/http"
	"net/url"
	"strings"
)

const jwplayerSegtax = 502
const jwplayerDomain = "jwplayer.com"

type JWContentMetadata struct {
	Url string
	Title string
	Description string
}

type JWContentExt struct {
	Description string `json:"description"`
}

type JWDataExt struct {
	Segtax int `json:"segtax"`
}

type JWTargetingResponse struct {
	Uuid string `json:"uuid"`
	Data JWTargetingData `json:"data"`
}

type JWTargetingData struct {
	MediaId string `json:"media_id"`
	BaseSegments []string `json:"base_segments"`
	TargetingProfiles []string `json:"targeting_profiles"`
}

func EnnrichRequest(request *openrtb2.BidRequest, publisherId string) {
	println("begin enrichment")
	if site := request.Site; site != nil {
		AddTargeting(&site.Keywords, site.Content, publisherId)
	}

	if app := request.App; app != nil {
		AddTargeting(&app.Keywords, app.Content, publisherId)
	}

	println("end enrichment")
}

func AddTargeting(keywords *string, content *openrtb2.Content, publisherId string) {
	jwpsegs := GetExistingJwpsegs(content.Data)
	if jwpsegs != nil && len(jwpsegs) > 0 {
		writeToKeywords(keywords, jwpsegs)
		return
	}

	if publisherId == "" {
		// error: missing pubid
		return
	}

	metadata := ParseContentMetadata(*content)
	if metadata.Url == "" {
		// error: missing media url
		return
	}

	channel := make(chan *JWTargetingResponse)
	FetchContentTargeting(publisherId, metadata, channel)
	targetingResponse := <- channel
	if targetingResponse == nil {
		return
	}

	jwpsegs = GetAllSegments(targetingResponse.Data)
	if len(jwpsegs) == 0 {
		// error: segments from req were empty
		return
	}

	writeToKeywords(keywords, jwpsegs)

	contentDatum := GetContentDatum(jwpsegs)
	content.Data = append(content.Data, contentDatum)
}

func writeToKeywords(keywords *string, segments []string) {
	if len(*keywords) > 0 {
		*keywords += ","
	}
	*keywords += GetKeywords(segments)
}

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

func FetchContentTargeting(publisherId string, contentMetadata JWContentMetadata, c chan *JWTargetingResponse) {
	mediaUrl := url.QueryEscape(contentMetadata.Url)
	title := url.QueryEscape(contentMetadata.Title)
	description := url.QueryEscape(contentMetadata.Description)
	reqUrl := fmt.Sprintf("https://content-targeting-api.longtailvideo.com/property/%s/content_segments?content_url=%s&title=%s&description=%s", publisherId, mediaUrl, title, description)
	resp, err := http.Get(reqUrl)
	if err != nil {
		fmt.Println("error: ", err)
		// error: request error
		return
	}

	defer resp.Body.Close()

	targetingResponse := JWTargetingResponse{}
	if error := json.NewDecoder(resp.Body).Decode(&targetingResponse); error != nil {
		// error: parsing error
		fmt.Println("error2: ", error)
		return
	}

	fmt.Println("targetingResponse: ", targetingResponse)
	c <- &targetingResponse
}

func GetAllSegments(targeting JWTargetingData) []string {
	return append(targeting.BaseSegments, targeting.TargetingProfiles...)
}

func GetKeywords(segments []string) string {
	if len(segments) == 0 {
		return ""
	}

	keyword := "jwpseg="
	// expected format: jwpseg=1,jwpseg=2,jwpseg=3
	keyword += strings.Join(segments, "," + keyword)
	return keyword
}

func GetContentDatum(segments []string) (contentData openrtb2.Data) {
	contentData.Name = jwplayerDomain
	contentData.Segment = GetContentSegments(segments)
	dataExt := JWDataExt{
		Segtax: jwplayerSegtax,
	}
	contentData.Ext, _ = json.Marshal(dataExt)
	return contentData
}

func GetContentSegments(rawSegments []string) []openrtb2.Segment {
	segments := make([]openrtb2.Segment, len(rawSegments))
	for _, rawSegment := range rawSegments {
		segment := openrtb2.Segment{
			Value: rawSegment,
		}
		segments = append(segments, segment)
	}

	return segments
}
