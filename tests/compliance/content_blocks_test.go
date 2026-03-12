package compliance

import (
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
)

// TestCompliance_ContentBlock_Text verifies text content block format compliance.
func TestCompliance_ContentBlock_Text(t *testing.T) {
	block := acp.TextBlock("Hello, world!")

	verifyContentBlock(t, block)
	assert.NotNil(t, block.Text, "Text block should have Text field")
	assert.Equal(t, "Hello, world!", block.Text.Text, "Text content should match")
}

// TestCompliance_ContentBlock_Image verifies image content block format compliance.
func TestCompliance_ContentBlock_Image(t *testing.T) {
	data := "base64imagedata123"
	mimeType := "image/png"
	block := acp.ImageBlock(data, mimeType)

	verifyContentBlock(t, block)
	assert.NotNil(t, block.Image, "Image block should have Image field")
	assert.Equal(t, data, block.Image.Data, "Image data should match")
	assert.Equal(t, mimeType, block.Image.MimeType, "MIME type should match")
}

// TestCompliance_ContentBlock_Audio verifies audio content block format compliance.
func TestCompliance_ContentBlock_Audio(t *testing.T) {
	data := "base64audiodata456"
	mimeType := "audio/mpeg"
	block := acp.AudioBlock(data, mimeType)

	verifyContentBlock(t, block)
	assert.NotNil(t, block.Audio, "Audio block should have Audio field")
	assert.Equal(t, data, block.Audio.Data, "Audio data should match")
	assert.Equal(t, mimeType, block.Audio.MimeType, "MIME type should match")
}

// TestCompliance_ContentBlock_ResourceLink verifies resource link content block format compliance.
func TestCompliance_ContentBlock_ResourceLink(t *testing.T) {
	name := "file.txt"
	uri := "file:///path/to/file.txt"
	block := acp.ResourceLinkBlock(name, uri)

	verifyContentBlock(t, block)
	assert.NotNil(t, block.ResourceLink, "Resource link block should have ResourceLink field")
	assert.Equal(t, uri, block.ResourceLink.Uri, "URI should match")
}

// TestCompliance_ContentBlock_Resource verifies embedded resource content block format compliance.
func TestCompliance_ContentBlock_Resource(t *testing.T) {
	// Text resource.
	textResource := acp.ContentBlock{
		Resource: &acp.ContentBlockResource{
			Resource: acp.EmbeddedResourceResource{
				TextResourceContents: &acp.TextResourceContents{
					Uri:  "file:///path/to/file.txt",
					Text: "file content",
				},
			},
		},
	}

	verifyContentBlock(t, textResource)
	assert.NotNil(t, textResource.Resource, "Resource block should have Resource field")
	assert.NotNil(t, textResource.Resource.Resource.TextResourceContents, "Text resource should have TextResourceContents")

	// Blob resource.
	blobResource := acp.ContentBlock{
		Resource: &acp.ContentBlockResource{
			Resource: acp.EmbeddedResourceResource{
				BlobResourceContents: &acp.BlobResourceContents{
					Uri:      "file:///path/to/file.bin",
					MimeType: stringPtr("application/octet-stream"),
					Blob:     "base64data",
				},
			},
		},
	}

	verifyContentBlock(t, blobResource)
	assert.NotNil(t, blobResource.Resource, "Resource block should have Resource field")
	assert.NotNil(t, blobResource.Resource.Resource.BlobResourceContents, "Blob resource should have BlobResourceContents")
}

// TestCompliance_ContentBlock_UTF8 verifies UTF-8 encoding compliance.
func TestCompliance_ContentBlock_UTF8(t *testing.T) {
	// Test with UTF-8 characters.
	utf8Text := "Hello, 世界! 🌍"
	block := acp.TextBlock(utf8Text)

	verifyContentBlock(t, block)
	assert.Equal(t, utf8Text, block.Text.Text, "UTF-8 text should be preserved")
}

// stringPtr returns a pointer to the string.
func stringPtr(s string) *string {
	return &s
}
