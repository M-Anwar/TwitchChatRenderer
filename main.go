package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/faiface/pixel/imdraw"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/g4s8/hexcolor"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
)

func loadTTF(path string, size float64) (font.Face, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	font, err := truetype.Parse(bytes)
	if err != nil {
		return nil, err
	}

	return truetype.NewFace(font, &truetype.Options{
		Size:              size,
		GlyphCacheEntries: 1,
	}), nil
}

func savePng(imgBytes []uint8, path string) error {
	myImage := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	myImage.Pix = imgBytes

	out, err := os.Create(path)

	if err != nil {
		return err
	}

	err = png.Encode(out, myImage)

	if err != nil {
		return fmt.Errorf("Unable to Encode to png: %s", err)
	}
	out.Close()
	return nil
}

type Config struct {
	Width, Height, DisplayWidth, DisplayHeight float64
	StartTime, EndTime                         float64
	Interactive                                bool
	ChatPath                                   string
	IsPreview                                  bool
	FPS                                        float64
	OutputPath                                 string
	HWAccel                                    bool
	Debug                                      bool
	ChatBounds                                 pixel.Rect
	FontSize                                   float64
	FontPath                                   string
}

func GetConfig() *Config {
	config := &Config{}
	var boundStr string
	flag.Float64Var(&config.Width, "width", 1920, "The width of the final video")
	flag.Float64Var(&config.Height, "height", 1080, "The height of the final video")
	flag.Float64Var(&config.DisplayWidth, "dwidth", 720, "The width of the interactive display")
	flag.Float64Var(&config.DisplayHeight, "dheight", 480, "The height of the interactive display")
	flag.BoolVar(&config.Interactive, "i", false, "Whether to show the results of the rendering to a screen at the desired FPS")
	flag.BoolVar(&config.IsPreview, "preview", false, "Preview only and not render to video")
	flag.Float64Var(&config.StartTime, "s", 0, "The start time to render the comments from")
	flag.Float64Var(&config.EndTime, "e", 10, "The end time to render the comments to")
	flag.StringVar(&config.ChatPath, "p", "sample_comments.csv", "The path with the chat data to render")
	flag.Float64Var(&config.FPS, "fps", 24.0, "The framerate at which to render the video")
	flag.StringVar(&config.OutputPath, "o", "sample.mov", "The file to output the video to")
	flag.BoolVar(&config.HWAccel, "hwaccel", false, "Use NVidia's NVENC encoder (only if FFMpeg supports it and alpha channel is not required)")
	flag.BoolVar(&config.Debug, "debug", false, "Print additional debug information from FFMpeg")
	flag.StringVar(&boundStr, "bounds", "0:0:200:200", "The bounds of where to draw the chat formated as x:y:width:height")
	flag.Float64Var(&config.FontSize, "font_size", 24, "The font size to use")
	flag.StringVar(&config.FontPath, "font_path", "Roboto-Regular.ttf", "The path to the ttf font to use")
	flag.Parse()

	boundParts := strings.Split(boundStr, ":")
	var boundFloats []float64
	if len(boundParts) != 4 {
		panic("Must have bounds specified with 4 variables, x:y:width:height")
	}
	for _, bound := range boundParts {
		boundInt, err := strconv.ParseFloat(bound, 64)
		if err != nil {
			panic(err)
		}
		boundFloats = append(boundFloats, boundInt)
	}

	config.ChatBounds = pixel.Rect{Min: pixel.V(boundFloats[0], boundFloats[1]), Max: pixel.V(boundFloats[0]+boundFloats[2], boundFloats[1]+boundFloats[3])}

	if config.EndTime <= config.StartTime {
		panic("Endtime must be larger than start time")
	}
	return config
}

func run() {

	// Load in Configs from command line
	config := GetConfig()
	fmt.Printf("%+v\n", config)

	// Load Comments to render
	comments, err := LoadCommentsFromCSV(config.ChatPath, config.StartTime, config.EndTime)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Loaded in %d comments within interval [%f, %f]\n", len(comments), config.StartTime, config.EndTime)

	// Load all emoticons in the chat
	emoteCache, err := NewEmoticonCache(comments)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Loaded in and generated emote atlas of %d emotes\n", emoteCache.Length())

	// FFMpeg Initialization
	var encodingStream chan []uint8
	var encodingDoneChan chan bool
	if !config.IsPreview {
		encodingStream = make(chan []uint8)
		encodingDoneChan = ffmpegEncode(
			encodingStream, config.OutputPath,
			int(config.FPS), int(config.Width), int(config.Height),
			config.HWAccel, config.Debug,
		)
	}

	// Create OpenGL Context and window with offscreen FBO (frame buffer object)
	cfg := pixelgl.WindowConfig{
		Title:     "Clipply Chat Renderer",
		Bounds:    pixel.R(0, 0, config.DisplayWidth, config.DisplayHeight),
		Resizable: true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}
	win.SetSmooth(true)
	renderingCanvas := pixelgl.NewCanvas(pixel.Rect{Min: pixel.V(0, 0), Max: pixel.V(config.Width, config.Height)})
	txtCanvas := pixelgl.NewCanvas(pixel.Rect{Min: pixel.V(0, 0), Max: pixel.V(config.ChatBounds.W(), config.ChatBounds.H())})
	renderingCanvas.SetSmooth(true)
	txtCanvas.SetSmooth(true)
	fmt.Println("Created Canvas")

	// Compute offscreen buffer scaling for rendering to live preview
	scaleX := config.DisplayWidth / config.Width
	scaleY := config.DisplayHeight / config.Height

	// Load in Font
	face, err := loadTTF(config.FontPath, config.FontSize)
	if err != nil {
		panic(err)
	}
	atlas := text.NewAtlas(face, text.ASCII)
	txt := text.New(pixel.V(0, 0), atlas)
	curTimeTxt := text.New(pixel.V(0, 0), atlas)
	curTimeTxt.Color = colornames.Orangered

	// Compute Rendering Variables
	totalTime := config.EndTime - config.StartTime
	frames := int(totalTime * config.FPS)
	deltaTime := 1 / config.FPS
	fmt.Printf("Total Frames: %d \t Total Time: %f\n", frames, totalTime)

	// Initialize Emoticon Rendering Batch
	spritesheet := emoteCache.GetSpriteSheet()
	batch := pixel.NewBatch(&pixel.TrianglesData{}, *spritesheet)

	// Initialize Chat background overlay
	imd := imdraw.New(nil)
	imd.Color = color.RGBA{0x00, 0x00, 0x00, 0x78}
	imd.Push(config.ChatBounds.Min.Sub(pixel.V(10, 10)), config.ChatBounds.Max.Add(pixel.V(10, 10)))
	imd.Rectangle(0)

	fpsTick := time.Tick(time.Second / 24)
	commentIdx := 0
	for frame := 0; frame < frames; frame++ {
		curTime := float64(frame)*deltaTime + config.StartTime
		fmt.Printf("\rFrame: %d \t Time: %f \t %f %%", frame+1, curTime, float64(frame+1)/float64(frames)*100)

		for idx := commentIdx; idx < len(comments); idx++ {
			if comments[idx].ContentOffsetSeconds <= curTime {
				hexColorStr := comments[idx].UserColor
				commentColor := colornames.Purple
				if hexColorStr != "" {
					commentColor, err = hexcolor.Parse(comments[idx].UserColor)
					if err != nil {
						commentColor = colornames.Purple
					}
				}

				// Render the commenters name
				txt.Color = commentColor
				txt.WriteString(fmt.Sprintf("%s:", comments[idx].CommenterDisplayName))
				txt.Color = colornames.White

				// Render word-wrapped chat message.
				lineString := comments[idx].CommenterDisplayName
				emoticonWidths := 0.0
				for _, mes := range comments[idx].MessageFragments {
					if mes.Emoticon.EmoticonID != "" {
						sprite, err := emoteCache.GetSprite(mes.Emoticon.EmoticonID)
						if err != nil {
							panic(err)
						}
						scale := txt.LineHeight * 0.85 / -sprite.Frame().H()
						spriteWidth := sprite.Frame().W() * scale
						spriteHeight := -sprite.Frame().H() * scale
						emoticonWidths += spriteWidth
						bounds := txt.BoundsOf(lineString)
						if bounds.W()+emoticonWidths > config.ChatBounds.W() {
							txt.WriteString("\n")
							emoticonWidths = spriteWidth
							lineString = ""
						}

						position := txt.Dot.Add(pixel.V(spriteWidth/2, spriteHeight/2))
						sprite.Draw(batch, pixel.IM.Scaled(pixel.ZV, scale).Moved(position))
						txt.Dot = txt.Dot.Add(pixel.V(spriteWidth, 0))
					} else {
						parts := strings.Split(mes.Text, " ")
						for _, part := range parts {
							bounds := txt.BoundsOf(lineString + part)

							if bounds.W()+emoticonWidths > config.ChatBounds.W() {
								txt.WriteString(fmt.Sprintf("\n%s ", part))
								lineString = part + " "
								emoticonWidths = 0
							} else {
								txt.WriteString(fmt.Sprintf("%s ", part))
								lineString += part + " "
							}
						}
					}
				}
				txt.WriteString("\n")

				commentIdx = idx + 1
			} else {
				break
			}
		}

		win.Clear(color.RGBA{0xFF, 0xFF, 0xFF, 0xFF})
		renderingCanvas.Clear(color.RGBA{0x00, 0x00, 0x00, 0x00})
		txtCanvas.Clear(color.RGBA{0x00, 0x00, 0x00, 0x00})

		// Draw the text to the canvas
		txt.Draw(txtCanvas, pixel.IM.Moved(pixel.V(0, txt.Bounds().H()-txt.LineHeight)))

		// We need to set the transform of the canvas before we draw our sprites so that they scroll as well
		txtCanvas.SetMatrix(pixel.IM.Moved(pixel.V(0, txt.Bounds().H()-txt.LineHeight)))
		batch.Draw(txtCanvas)
		txtCanvas.SetMatrix(pixel.IM)

		// Finally draw the text canvas to the rendering canvas
		imd.Draw(renderingCanvas)
		txtCanvas.Draw(renderingCanvas, pixel.IM.Moved(config.ChatBounds.Center()))
		renderingCanvas.Draw(win, pixel.IM.ScaledXY(pixel.V(0, 0), pixel.V(scaleX, scaleY)).Moved(win.Bounds().Center()))

		curTimeTxt.Clear()
		fmt.Fprintf(curTimeTxt, "Current Time: %f", curTime)
		curTimeTxt.Draw(win, pixel.IM.Scaled(pixel.V(0, 0), 0.5).Moved(pixel.V(0, win.Bounds().H()-curTimeTxt.Bounds().H()/2)))
		win.Update()

		if !config.IsPreview {
			encodingStream <- renderingCanvas.Pixels()
		}
		if config.Interactive {
			<-fpsTick
		}

	}
	fmt.Println("\nRendering Sequence Complete")
	if !config.IsPreview {
		close(encodingStream)
		<-encodingDoneChan
		fmt.Println("Rendering Complete!")
	}
}

func main() {
	pixelgl.Run(run)
}
