package jwplayer

import (
	"github.com/prebid/openrtb/v20/adcom1"
	"net/url"
)

func IsInstream(placementType adcom1.VideoPlacementSubtype) bool {
	return placementType == adcom1.VideoInStream
}

func IsValidMediaUrl(rawUrl string) bool {
	parsedUrl, error := url.ParseRequestURI(rawUrl)
	if error != nil {
		return false
	}

	isLocalFile := parsedUrl.Scheme == "file"
	isLocalHost := parsedUrl.Opaque != ""

	if isLocalFile || isLocalHost {
		return false
	}

	path := parsedUrl.Path
	isRelativePath := parsedUrl.Scheme == "" && len(path) > 2 && path[0:1] == "/" && path[1:2] != "/"

	return !isRelativePath
}

func Map[T any, M any](inputs []T, f func(T) M) []M {
	outputs := make([]M, len(inputs))
	for i, element := range inputs {
		outputs[i] = f(element)
	}
	return outputs
}
