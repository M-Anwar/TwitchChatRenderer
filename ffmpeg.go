package main

import (
	"bufio"
	"fmt"
	"os/exec"
)

func getFFMPEGVersion() (string, error) {
	cmd := exec.Command("ffmpeg", "-version")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	buf := bufio.NewReader(stdout)
	version, _, err := buf.ReadLine()

	if err != nil {
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	return string(version), nil
}

func ffmpegEncode(inputStream <-chan []uint8, outFile string, FPS int, width int, height int, hwAccel bool, debug bool) chan bool {
	done := make(chan bool, 1)
	go func() {
		fmt.Println("Running encoding thread")
		var args []string
		args = append(args, "-y")                                      // Overwrite output
		args = append(args, "-f", "rawvideo", "-pix_fmt", "rgba")      // Input will be rawvideo of rgba format
		args = append(args, "-r", fmt.Sprintf("%d", FPS))              // Set the FPS of the input
		args = append(args, "-s", fmt.Sprintf("%dx%d", width, height)) // Set the dimensions of the input video
		args = append(args, "-i", "pipe:0")                            // Set the input to come from stdin
		if hwAccel {
			fmt.Println("Using Hardware Acceleration")
			args = append(args, "-c:v", "h264_nvenc", "-preset", "fast") // If we have hardware acceleration turned on use NVidia's NVENC
		} else {
			args = append(args, "-pix_fmt", "yuva444p10le") // The output video should have YUV pixel color space
			args = append(args, "-c:v", "qtrle")            // Use the quicktime codec for transparancy support
			args = append(args, "-preset", "ultrafast")     // Use the fastest preset possible
		}
		args = append(args, "-vf", "vflip") // Flip the output video along the vertical axis (the OpenGL buffer returns flipped data)
		args = append(args, outFile)        // Write to the specific output file

		cmd := exec.Command("ffmpeg", args...)

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return
		}

		if debug {
			stderr, err := cmd.StderrPipe()
			if err != nil {
				return
			}
			go func() {
				rd := bufio.NewReader(stderr)
				for {
					str, err := rd.ReadString('\n')
					if err != nil {
						fmt.Println(err)
						return
					}
					fmt.Println(str)
				}
			}()
		}

		err = cmd.Start()
		if err != nil {
			fmt.Println(err)
			return
		}

		for frame := range inputStream {
			stdin.Write(frame)
		}
		stdin.Close()

		fmt.Println("Waiting for ffmpeg to finish")
		err = cmd.Wait()

		if err != nil {
			fmt.Printf("Error During FFMPEG encoding: %s\n", err)
			return
		}
		close(done)

	}()

	return done
}
