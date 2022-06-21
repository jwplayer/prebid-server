package jwplayer

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContentMetadataParseSuccess(t *testing.T) {
	description := "Test Description"
	descriptionExt, _ := json.Marshal("{\"description\": \"Test Description\"}")
	content := openrtb2.Content{
		Url:   "//test.medial.url/file.mp4",
		Title: "Test title",
		Ext:   descriptionExt,
	}

	metadata := ParseContentMetadata(content)
	assert.Equal(t, content.Url, metadata.Url)
	assert.Equal(t, content.Title, metadata.Title)
	assert.Equal(t, description, metadata.Description)
}
