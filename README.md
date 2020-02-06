# TwitchChatRenderer
Renders Twitch Chat to a Video that can be used as a Composite.

## How to Build
```
git clone https://github.com/M-Anwar/TwitchChatRenderer.git
cd TwitchChatRenderer
go build
```

## How to Run (TODO)
On windows:
```
ChatRendering.exe -h
```
Will print out information on how to run the application:
```
Usage of ChatRendering.exe:
  -bounds string
        The bounds of where to draw the chat formated as x:y:width:height (default "0:0:200:200")
  -debug
        Print additional debug information from FFMpeg
  -dheight float
        The height of the interactive display (default 480)
  -dwidth float
        The width of the interactive display (default 720)
  -e float
        The end time to render the comments to (default 10)
  -font_path string
        The path to the ttf font to use (default "Roboto-Regular.ttf")
  -font_size float
        The font size to use (default 24)
  -fps float
        The framerate at which to render the video (default 24)
  -height float
        The height of the final video (default 1080)
  -hwaccel
        Use NVidia's NVENC encoder (only if FFMpeg supports it and alpha channel is not required)
  -i    Whether to show the results of the rendering to a screen at the desired FPS
  -o string
        The file to output the video to (default "sample.mov")
  -p string
        The path with the chat data to render (default "sample_comments.csv")
  -preview
        Preview only and not render to video
  -s float
        The start time to render the comments from
  -width float
        The width of the final video (default 1920)
```