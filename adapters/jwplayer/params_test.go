package jwplayer

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// This file actually intends to test static/bidder-params/jwplayer.json
//
// These also validate the format of the external API: request.imp[i].ext.jwplayer

// TestValidParams makes sure that the jwplayer schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator := getValidator(t)
	for _, validParam := range validParams {
		fmt.Println(validParam)
		if err := validator.Validate(openrtb_ext.BidderJWPlayer, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected jwplayer params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the sonobi schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator := getValidator(t)
	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderJWPlayer, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

func getValidator(t *testing.T) openrtb_ext.BidderParamValidator {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	return validator
}

var validParams = []string{
	`{"placementId": "123"}`,
	`{"placementId": "123", "publisherId": "abc"}`,
}

var invalidParams = []string{
	`{"tagId": "abc"}`,
	`{"publisherId": "abc"}`,
}
