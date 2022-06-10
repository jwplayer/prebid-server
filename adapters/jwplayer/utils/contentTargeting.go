package utils

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"net/http"
	"net/url"
	"strings"
)

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
	metadata := GetContentMetadata(content)
	targetingResponse := GetContentTargeting(publisherId, metadata)
	if targetingResponse == nil {
		return
	}

	segments := GetAllSegments(targetingResponse.Data)
	if len(segments) == 0 {
		return
	}

	if len(*keywords) > 0 {
		*keywords += ","
	}
	*keywords += GetKeywords(segments)

	contentDatum := GetContentDatum(segments)
	content.Data = append(content.Data, contentDatum)
}

func GetContentMetadata(content *openrtb2.Content) JWContentMetadata {
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

func GetContentTargeting(publisherId string, contentMetadata JWContentMetadata) *JWTargetingResponse {
	// http get
	// c <- json

	mediaUrl := url.QueryEscape(contentMetadata.Url)
	title := url.QueryEscape(contentMetadata.Title)
	description := url.QueryEscape(contentMetadata.Description)
	reqUrl := fmt.Sprintf("https://content-targeting-api.longtailvideo.com/property/%s/content_segments?content_url=%s&title=%s&description=%s", publisherId, mediaUrl, title, description)
	resp, err := http.Get(reqUrl)
	if err != nil {
		fmt.Println("error: ", err)
		return nil
	}

	defer resp.Body.Close()

	targetingResponse := JWTargetingResponse{}
	if error := json.NewDecoder(resp.Body).Decode(&targetingResponse); error != nil {
		fmt.Println("error2: ", error)
		return nil
	}

	fmt.Println("targetingResponse: ", targetingResponse)

	return &targetingResponse
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
	contentData.Name = "jwplayer.com"
	contentData.Segment = GetContentSegments(segments)
	dataExt := JWDataExt{
		Segtax: 502,
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
