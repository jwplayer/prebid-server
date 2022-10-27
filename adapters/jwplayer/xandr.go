package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v16/adcom1"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"strconv"
	"strings"
)

// copied from appnexus.go appnexusImpExtAppnexus
type xandrImpExtParams struct {
	PlacementID int `json:"placement_id,omitempty"`
}

// copied from appnexus.go appnexusImpExt
type xandrImpExt struct {
	Appnexus xandrImpExtParams `json:"appnexus"`
}

func GetXandrImpExt(placementId string) json.RawMessage {
	id, conversionError := strconv.Atoi(placementId)
	if conversionError != nil {
		return nil
	}

	appnexusExt := &xandrImpExt{
		Appnexus: xandrImpExtParams{
			PlacementID: id,
		},
	}

	jsonExt, jsonError := json.Marshal(appnexusExt)
	if jsonError != nil {
		return nil
	}

	return jsonExt
}

type xandrVideoExt struct {
	Appnexus xandrVideoExtParams `json:"appnexus"`
}

type xandrVideoExtParams struct {
	Context xandrContext `json:"context,omitempty"`
}

type xandrContext int

const (
	Unknown   xandrContext = 0
	PreRoll   xandrContext = 1
	MidRoll   xandrContext = 2
	PostRoll  xandrContext = 3
	Outstream xandrContext = 4
)

func SetXandrVideoExt(video *openrtb2.Video) {
	context := GetXandrContext(*video)
	if context == Unknown {
		return
	}

	videoExt := xandrVideoExt{
		Appnexus: xandrVideoExtParams{
			Context: context,
		},
	}
	video.Ext, _ = json.Marshal(videoExt)
}

func GetXandrContext(video openrtb2.Video) xandrContext {
	if video.Placement == 0 {
		return Unknown
	}

	if IsInstream(video.Placement) == false {
		return Outstream
	}

	if video.StartDelay == nil {
		return Unknown
	}

	return GetXandrContextFromStartdelay(*video.StartDelay)
}

func GetXandrContextFromStartdelay(startDelay adcom1.StartDelay) xandrContext {
	if startDelay > 0 {
		return MidRoll // startdelay > 0 indicates ad position in seconds
	}

	switch startDelay {
	case adcom1.StartPreRoll:
		return PreRoll
	case adcom1.StartMidRoll:
		return MidRoll
	case adcom1.StartPostRoll:
		return PostRoll
	}

	return Unknown
}

func WriteToXandrKeywords(keywords *string, jwpsegs []string) {
	if len(jwpsegs) == 0 {
		return
	}

	if len(*keywords) > 0 {
		*keywords += ","
	}

	jwpsegToKeyword := func(jwpseg string) string { return "jwpseg=" + jwpseg }
	newKeywords := Map(jwpsegs, jwpsegToKeyword)
	*keywords += strings.Join(newKeywords, ",")
}

func ConvertToXandrKeywords(jwpsegs []string) string {
	if len(jwpsegs) == 0 {
		return ""
	}

	keyword := "jwpseg="
	// expected format: jwpseg=1,jwpseg=2,jwpseg=3
	keyword += strings.Join(jwpsegs, ","+keyword)
	return keyword
}

func GetXandrRequestExt(schain openrtb2.SupplyChain) json.RawMessage {
	// Xandr expects the SChain to be in accordance with oRTB 2.4
	// $.ext.schain
	requestExtension := requestExt{
		SChain: schain,
	}
	jsonExt, jsonError := json.Marshal(requestExtension)
	if jsonError != nil {
		return nil
	}
	return jsonExt
}
