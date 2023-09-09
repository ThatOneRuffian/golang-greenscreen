package main

import (
	"fmt"
	"image"
	"time"

	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"gocv.io/x/gocv"
)

func init() {
	// TODO  create slice of capture devices with active/inactive status for GUI
	// --------- init media output dir
	if err := initOutputDir(); err != nil {
		panic(err)
	}
}

func main() {

	// --------- init webcam
	camera1 := captureDevice{
		deviceID:      0,
		frameRate:     24.0,
		captureHeight: 480.0,
		captureWidth:  864.0,
	}
	if err := camera1.initCaptureDevice(); err != nil {
		fmt.Printf("Issue Opening Capture Device %d \n", camera1.deviceID)
	}
	defer camera1.captureDevice.Close()

	// --------- init background image/video
	// Load the background image
	backgroundImage := &inputImage{
		sourceFile:  "./background.jpg",
		frameBuffer: new(gocv.Mat),
	}
	defer backgroundImage.frameBuffer.Close()
	// todo make init to load image and handle error
	// resize to canvas size on init
	*backgroundImage.frameBuffer = gocv.IMRead(backgroundImage.sourceFile, gocv.IMReadColor)

	// Resize the background image to match the frame size
	gocv.Resize(*backgroundImage.frameBuffer, backgroundImage.frameBuffer, image.Point{int(camera1.captureWidth), int(camera1.captureHeight)}, 0, 0, gocv.InterpolationDefault)

	// Load background video background.mp4
	backgroundVideo := &inputVideo{
		sourceFile:  "./background.mp4",
		frameBuffer: new(gocv.Mat),
	}
	var vidErr error
	backgroundVideo.videoReader, vidErr = gocv.VideoCaptureFile(backgroundVideo.sourceFile)
	if vidErr != nil {
		panic(fmt.Sprintf("Could Not Open Background Video. %v", backgroundVideo.sourceFile))
	}
	defer backgroundVideo.videoReader.Close()
	defer backgroundVideo.frameBuffer.Close()
	*backgroundVideo.frameBuffer = gocv.NewMat()

	// ------------ init stream writers
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

	// ------------ main display window loop
	fyneApp := app.New()
	fyneWindow := fyneApp.NewWindow("Stream")
	defer fyneWindow.Close()

	fyneImg, _ := getNextBackgroundBuffer(backgroundImage).ToImage()

	fyneImage := canvas.NewImageFromImage(fyneImg)
	fyneImage.FillMode = canvas.ImageFillOriginal
	fyneWindow.SetContent(container.NewMax(fyneImage))

	// app loop for video rendering
	ticker := time.NewTicker(30 * time.Millisecond)

	go func(backgroundFeed backgroundStream) {
		for {
			select {
			case <-ticker.C:
				if !camera1.nextFrame() {
					fmt.Println("Error Fetching Frame From Capture Device.")
					break
				}
				// record raw capture stream to file
				rawWriter.Write(*camera1.frameBuffer)

				nextFrame := getNextBackgroundBuffer(backgroundFeed)
				if nextFrame == nil {
					fmt.Println("Issue getting background image frame buffer")
				} else if nextFrame.Cols() != camera1.frameBuffer.Cols() || nextFrame.Rows() != camera1.frameBuffer.Rows() {
					// TODO this overwrites the buffer for nextFrame
					// this size should be set on init and done auto on getFrame - should prob be based on the canvas size type? on init?
					gocv.Resize(*nextFrame, nextFrame, image.Point{camera1.frameBuffer.Cols(), camera1.frameBuffer.Rows()}, 0, 0, gocv.InterpolationDefault)
				}

				fxImg := gocv.NewMat()
				defer fxImg.Close()

				// add green screen mask effect to stream frame, exposing background
				addGreenScreenMask(camera1.frameBuffer, nextFrame, &fxImg)
				canvasImg, _ := fxImg.ToImage()

				// write fx video frame to disk
				fxWriter.Write(fxImg)
				fxImg.Close()

				// update fyne image canvas
				fyneImage.Image = canvasImg
				fyneImage.Refresh()
			}
		}
	}(backgroundVideo)

	// run fyne app
	// TODO ESC to shutdown program?
	fyneWindow.ShowAndRun()
	fyneWindow.Close()
}
