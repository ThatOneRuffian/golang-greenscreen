package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

var Window = gocv.NewWindow("Feed Preview")

func main() {
	defer Window.Close()
	// --------- init webcam
	camera1 := captureDevice{
		deviceID:      0,
		frameRate:     24.0,
		captureHeight: 480.0,
		captureWidth:  864.0,
	}
	if err := camera1.initCaptureDevice(); err != nil {
		fmt.Printf("Issue Opening Capture Device %d", camera1.deviceID)
	}
	defer camera1.captureDevice.Close()

	// Load the background image
	backgroundPath := "./background.jpg" // Specify the path to your background image
	background := gocv.IMRead(backgroundPath, gocv.IMReadColor)
	defer background.Close()

	// Resize the background image to match the frame size
	gocv.Resize(background, &background, image.Point{int(camera1.captureWidth), int(camera1.captureHeight)}, 0, 0, gocv.InterpolationDefault)

	// Load background video background.mp4
	backgroundVideoPath := "./background.mp4" // Specify the path to your background video
	backgroundVideo, err := gocv.VideoCaptureFile(backgroundVideoPath)
	if err != nil {
		fmt.Printf("Error opening video file: %v\n", err)
		return
	}
	defer backgroundVideo.Close()

	backgroundFrame := gocv.NewMat()
	defer backgroundFrame.Close()

	// create writer for raw stream
	// todo the output dir should be made by here...
	rawSaveFile := fmt.Sprintf("%s/stream_raw_output.mp4", defaultOutputDir)
	rawWriter, err := gocv.VideoWriterFile(rawSaveFile, "mp4v", camera1.frameRate, int(camera1.captureWidth), int(camera1.captureHeight), true)
	if err != nil {
		fmt.Printf("Error opening video writer device: %v\n", rawSaveFile)
		fmt.Printf("err: %v\n", err)
		return
	}
	defer rawWriter.Close()

	// create writer for VFX stream
	fxSaveFile := fmt.Sprintf("%s/stream_fx_output.mp4", defaultOutputDir)
	fxWriter, err := gocv.VideoWriterFile(fxSaveFile, "mp4v", camera1.frameRate, int(camera1.captureWidth), int(camera1.captureHeight), true)
	if err != nil {
		fmt.Printf("Error opening FX video writer device: %v\n", fxSaveFile)
		fmt.Printf("err: %v\n", err)
		return
	}
	defer fxWriter.Close()

	for {
		// capture next video frame from webcam
		if !camera1.getNextFrame() {
			fmt.Println("Error Fetching Frame From Capture Device.")
			break
		}

		// record raw capture stream to file
		rawWriter.Write(*camera1.frameBuffer)

		// capture next video frame from file
		if ok := backgroundVideo.Read(&backgroundFrame); !ok {
			// attempt to set video file to first frame for EOF condition
			backgroundVideo.Set(gocv.VideoCapturePosFrames, 0)
			if ok := backgroundVideo.Read(&backgroundFrame); !ok {
				fmt.Println("An unkown error has occured while reading the provided background video.")
				break
			}
		}
		if backgroundFrame.Empty() {
			continue
		}

		// resize background video frame to match capture image
		gocv.Resize(backgroundFrame, &backgroundFrame, image.Point{camera1.frameBuffer.Cols(), camera1.frameBuffer.Rows()}, 0, 0, gocv.InterpolationDefault)

		fxImg := gocv.NewMat()
		defer fxImg.Close()

		// add green screen mask effect to stream frame, exposing background
		addGreenScreenMask(camera1.frameBuffer, &backgroundFrame, &fxImg)

		// write fx video frame
		fxWriter.Write(fxImg)

		// Update window
		Window.IMShow(fxImg)
		fxImg.Close()

		// ESC to shutdown program
		if Window.WaitKey(1) == 27 {
			break
		}
	}
}
