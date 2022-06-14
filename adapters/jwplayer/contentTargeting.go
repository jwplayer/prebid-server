package jwplayer

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
	Uuid string          `json:"uuid"`
	Data JWTargetingData `json:"data"`
}

type JWTargetingData struct {
	MediaId string `json:"media_id"`
	BaseSegments []string `json:"base_segments"`
	TargetingProfiles []string `json:"targeting_profiles"`
}

type requestEnricher struct {
	httpClient *http.Client
}

func BuildRequestEnricher(httpClient *http.Client) *requestEnricher {
	return &requestEnricher{
		httpClient: httpClient,
	}
}

func (enricher *requestEnricher) EnrichRequest(request *openrtb2.BidRequest, publisherId string) *TargetingFailed {
	if site := request.Site; site != nil {
		return enricher.enrich(&site.Keywords, site.Content, publisherId)
	}

	if app := request.App; app != nil {
		return enricher.enrich(&app.Keywords, app.Content, publisherId)
	}

	return nil
}

func (enricher *requestEnricher) enrich(keywords *string, content *openrtb2.Content, publisherId string) *TargetingFailed {
	jwpsegs := GetExistingJwpsegs(content.Data)
	if jwpsegs != nil && len(jwpsegs) > 0 {
		writeToKeywords(keywords, jwpsegs)
		return nil
	}

	if publisherId == "" {
		return &TargetingFailed{
			Message: "Missing PublisherId",
			code: MissingPublisherIdErrorCode,
		}
	}

	metadata := ParseContentMetadata(*content)
	if metadata.Url == "" {
		return &TargetingFailed{
			Message: "Missing Media Url",
			code: MissingMediaUrlErrorCode,
		}
	}

	targetingResponse, err := enricher.FetchContentTargeting(publisherId, metadata)
	if err != nil {
		return err
	}

	jwpsegs = GetAllJwpsegs(targetingResponse.Data)
	if len(jwpsegs) == 0 {
		return &TargetingFailed{
			Message: "Empty Targeting Segments",
			code: EmptyTargetingSegments,
		}
	}

	writeToKeywords(keywords, jwpsegs)

	contentDatum := MakeOrtbDatum(jwpsegs)
	content.Data = append(content.Data, contentDatum)

	return nil
}

func (enricher *requestEnricher) FetchContentTargeting(publisherId string, contentMetadata JWContentMetadata) (*JWTargetingResponse, *TargetingFailed) {
	mediaUrl := url.QueryEscape(contentMetadata.Url)
	title := url.QueryEscape(contentMetadata.Title)
	description := url.QueryEscape(contentMetadata.Description)

	reqUrl := fmt.Sprintf("https://content-targeting-api.longtailvideo.com/property/%s/content_segments?content_url=%s&title=%s&description=%s", publisherId, mediaUrl, title, description)
	httpReq, newReqErr := http.NewRequest("GET", reqUrl, nil)
	if newReqErr != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Failed to instantiate request: %s", newReqErr.Error()),
			code: HttpRequestInstantiationErrorCode,
		}
	}

	resp, reqErr := enricher.httpClient.Do(httpReq)

	if reqErr != nil {
		statusCode := resp.StatusCode
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Server responded with failure status: %d.", statusCode),
			code: BaseNetworkErrorCode + statusCode,
		}
	}

	defer resp.Body.Close()

	targetingResponse := JWTargetingResponse{}
	if error := json.NewDecoder(resp.Body).Decode(&targetingResponse); error != nil {
		return nil, &TargetingFailed{
			Message: fmt.Sprintf("Failed to decode targeting response: %s", error.Error()),
			code: BaseDecodingErrorCode,
		}
	}

	fmt.Println("targeting resposne: ", targetingResponse)
	return &targetingResponse, nil
}
