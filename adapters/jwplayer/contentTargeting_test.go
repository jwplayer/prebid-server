package jwplayer

import (
	"fmt"
	"testing"
)

func TestDemo(t *testing.T) {
	//D9hUeD6O
	//
	metadata := JWContentMetadata{
		Url: "http://example.com/media.mp4",
		Title: "Sample",
		Description: "desc",
	}
	resp, _ := FetchContentTargeting("D9hUeD6O", metadata)
	fmt.Println(resp)
}
