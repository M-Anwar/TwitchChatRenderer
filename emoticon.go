package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"os"

	"github.com/faiface/pixel"
)

type EmoticonData struct {
	URL    string
	Text   string
	Img    image.Image
	Bounds pixel.Rect
}

type EmoticonCache struct {
	cache            map[string]*EmoticonData
	spriteSheet      image.Image
	pixelSpriteSheet pixel.Picture
}

func downloadImage(url string) (image.Image, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	img, _, err := image.Decode(response.Body)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// generateSpriteSheet takes an emoticon cache and generates a single texture atlas
// with all the emoticons. The algorithm is a brute force scanline texture packer.
// Pick an initial size for the atlas, and try to fit all the images. If we run out of space
// resize the atlas in a power of 2, and try fitting all the images again. Repeat until we find a
// size that can fit everything. The images are packed from left to right.
func generateSpriteSheet(cache map[string]*EmoticonData) (image.Image, error) {
	initialW, initialH := 256, 256 // Choose a reasonable minimum atlas size
	curW, curH := initialW, initialH
	fit := false

	for fit == false { // Keep looping until all the emoticons have been fit
		fit = true
		spriteMap := image.NewRGBA(image.Rect(0, 0, curW, curH))
		x, y := 0, 0
		maxY := -1

	FitLoop:
		for k, v := range cache {

			if x+v.Img.Bounds().Dx() > curW { // If we have reached the far right, go back to the left with another row
				x = 0
				y += maxY
				maxY = -1
				if y >= curH { // If the new row doesn't fit within the image, we need to resize
					fit = false
					break FitLoop
				}
			}

			minX, minY := x, y
			maX, maY := x+v.Img.Bounds().Dx(), y+v.Img.Bounds().Dy()

			draw.Draw(spriteMap, image.Rect(minX, minY, maX, maY), v.Img, image.Point{0, 0}, draw.Src)
			cache[k].Bounds = pixel.Rect{Min: pixel.V(float64(minX), float64(curH-minY)), Max: pixel.V(float64(maX), float64(curH-maY))}

			x += v.Img.Bounds().Dx()
			if v.Img.Bounds().Dy() > maxY {
				maxY = v.Img.Bounds().Dy()
			}

		}
		if fit == true {
			return spriteMap, nil
		}
		curW *= 2
		curH *= 2
	}
	return nil, fmt.Errorf("Unable to generate sprite map")
}

func NewEmoticonCache(comments []*Message) (*EmoticonCache, error) {
	cache := make(map[string]*EmoticonData)

	// Populate Initial cache with all unique emoticons
	for _, comment := range comments {
		for _, frag := range comment.MessageFragments {
			if frag.Emoticon.EmoticonID != "" {
				_, ok := cache[frag.Emoticon.EmoticonID]
				if !ok {
					url := fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%s/1.0", frag.Emoticon.EmoticonID)
					img, err := downloadImage(url)
					if err != nil {
						return nil, err
					}
					cache[frag.Emoticon.EmoticonID] = &EmoticonData{
						URL:  url,
						Text: frag.Text,
						Img:  img,
					}
				}
			}
		}
	}

	spriteMap, err := generateSpriteSheet(cache)
	if err != nil {
		return nil, err
	}

	return &EmoticonCache{
		cache:            cache,
		spriteSheet:      spriteMap,
		pixelSpriteSheet: pixel.PictureDataFromImage(spriteMap),
	}, nil
}

func (e *EmoticonCache) SaveSpriteMap(path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	err = png.Encode(out, e.spriteSheet)
	if err != nil {
		return err
	}
	return nil
}

func (e *EmoticonCache) GetSpriteSheet() *pixel.Picture {
	return &e.pixelSpriteSheet
}

func (e *EmoticonCache) Length() int {
	return len(e.cache)
}

func (e *EmoticonCache) GetSprite(id string) (*pixel.Sprite, error) {
	elem, ok := e.cache[id]
	if !ok {
		return nil, fmt.Errorf("Unable to find emote with id %s", id)
	}
	return pixel.NewSprite(e.pixelSpriteSheet, elem.Bounds), nil
}
