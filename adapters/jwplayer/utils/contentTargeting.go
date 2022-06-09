package utils

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"net/http"
	"time"
)

type JWContentMetadata struct {
	url string
	title string
	description string
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

func EnnrichRequest(request *openrtb2.BidRequest) {
	println("begin enrichment")
	time.Sleep(8 * time.Second)
	println("end enrichment")
}

func GetContentTargeting(publisherId string, contentUrl string, title string, description string) *JWTargetingResponse {
	// http get
	// c <- json

	url := fmt.Sprintf("https://content-targeting-api.longtailvideo.com/property/%s/content_segments?content_url=%s&title=%s&description=%s", publisherId, contentUrl, title, description)
	resp, err := http.Get(url)
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

func getKeywords() {}


//func requestSegment
//


// request content targeting -> return body
// convert body to appnexus format
// write to request:
