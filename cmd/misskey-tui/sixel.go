package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"

	_ "github.com/gen2brain/webp"

	"github.com/mattn/go-sixel"
	"golang.org/x/image/draw"
)

func downloadAndEncode(client *http.Client, proxyURL string, imageURL string) ([]byte, error) {
	// Build proxy URL
	proxyReqURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy url: %w", err)
	}
	q := proxyReqURL.Query()
	q.Set("url", imageURL)
	q.Set("type", "emoji")
	proxyReqURL.RawQuery = q.Encode()

	// Download image from proxy
	resp, err := client.Get(proxyReqURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %s", resp.Status)
	}

	// Decode image
	src, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize image
	dst := image.NewRGBA(image.Rect(0, 0, 16, 16))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	// Encode to sixel
	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	enc.Height = 16
	enc.Width = 16
	err = sixel.NewEncoder(&buf).Encode(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to encode sixel: %w", err)
	}

	return buf.Bytes(), nil
}
