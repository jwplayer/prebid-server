package utils

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"net/http"
	"net/url"
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
	if site := request.Site; site != nil {
		Enrich(&site.Keywords, site.Content, publisherId)
	}

	if app := request.App; app != nil {
		Enrich(&app.Keywords, app.Content, publisherId)
	}
}

func Enrich(keywords *string, content *openrtb2.Content, publisherId string) {
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

	jwpsegs = GetAllJwpsegs(targetingResponse.Data)
	if len(jwpsegs) == 0 {
		// error: segments from req were empty
		return
	}

	writeToKeywords(keywords, jwpsegs)

	contentDatum := MakeOrtbDatum(jwpsegs)
	content.Data = append(content.Data, contentDatum)
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

func writeToKeywords(keywords *string, jwpsegs []string) {
	if len(*keywords) > 0 {
		*keywords += ","
	}
	*keywords += GetXandrKeywords(jwpsegs)
}
