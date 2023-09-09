package main

import (
	"fmt"
	"image"
	"time"

	"golang_greenscreen/streams"

	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"gocv.io/x/gocv"
)

func init() {
	// TODO  create slice of capture devices with active/inactive status for GUI
	// --------- init media output dir
	if err := streams.InitOutputDir(); err != nil {
		panic(err)
	}
}

func main() {

	// --------- init webcam
	camera1 := streams.CaptureDevice{
		DeviceID:      0,
		FrameRate:     24.0,
		CaptureHeight: 480.0,
		CaptureWidth:  864.0,
	}
	if err := camera1.InitCaptureDevice(); err != nil {
		fmt.Printf("Issue Opening Capture Device %d \n", camera1.DeviceID)
	}
	defer camera1.CaptureDevice.Close()

	// --------- init background image/video
	// Load the background image
	backgroundImage := &streams.InputImage{
		SourceFile:  "./background.jpg",
		FrameBuffer: new(gocv.Mat),
	}
	defer backgroundImage.FrameBuffer.Close()
	// todo make init to load image and handle error
	// resize to canvas size on init
	*backgroundImage.FrameBuffer = gocv.IMRead(backgroundImage.SourceFile, gocv.IMReadColor)

	// Resize the background image to match the frame size
	gocv.Resize(*backgroundImage.FrameBuffer, backgroundImage.FrameBuffer, image.Point{int(camera1.CaptureWidth), int(camera1.CaptureHeight)}, 0, 0, gocv.InterpolationDefault)

	// Load background video background.mp4
	backgroundVideo := &streams.InputVideo{
		SourceFile:  "./background.mp4",
		FrameBuffer: new(gocv.Mat),
	}
	var vidErr error
	backgroundVideo.VideoReader, vidErr = gocv.VideoCaptureFile(backgroundVideo.SourceFile)
	if vidErr != nil {
		panic(fmt.Sprintf("Could Not Open Background Video. %v", backgroundVideo.SourceFile))
	}
	defer backgroundVideo.VideoReader.Close()
	defer backgroundVideo.FrameBuffer.Close()
	*backgroundVideo.FrameBuffer = gocv.NewMat()

	// ------------ init stream writers
	// create writer for raw stream
	// todo the output dir should be made by here...
	rawSaveFile := fmt.Sprintf("%s/stream_raw_output.mp4", streams.DefaultOutputDir)
	rawWriter, err := gocv.VideoWriterFile(rawSaveFile, "mp4v", camera1.FrameRate, int(camera1.CaptureWidth), int(camera1.CaptureHeight), true)
	if err != nil {
		fmt.Printf("Error opening video writer device: %v\n", rawSaveFile)
		fmt.Printf("err: %v\n", err)
		return
	}
	defer rawWriter.Close()

	// create writer for VFX stream
	fxSaveFile := fmt.Sprintf("%s/stream_fx_output.mp4", streams.DefaultOutputDir)
	fxWriter, err := gocv.VideoWriterFile(fxSaveFile, "mp4v", camera1.FrameRate, int(camera1.CaptureWidth), int(camera1.CaptureHeight), true)
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

	fyneImg, _ := streams.GetNextBackgroundBuffer(backgroundImage).ToImage()

	fyneImage := canvas.NewImageFromImage(fyneImg)
	fyneImage.FillMode = canvas.ImageFillOriginal
	fyneWindow.SetContent(container.NewMax(fyneImage))

	// app loop for video rendering
	ticker := time.NewTicker(30 * time.Millisecond)

	go func(backgroundFeed streams.BackgroundStream) {
		for {
			select {
			case <-ticker.C:
				if !camera1.NextFrame() {
					fmt.Println("Error Fetching Frame From Capture Device.")
					break
				}
				// record raw capture stream to file
				rawWriter.Write(*camera1.FrameBuffer)

				nextFrame := streams.GetNextBackgroundBuffer(backgroundFeed)
				if nextFrame == nil {
					fmt.Println("Issue getting background image frame buffer")
				} else if nextFrame.Cols() != camera1.FrameBuffer.Cols() || nextFrame.Rows() != camera1.FrameBuffer.Rows() {
					// TODO this overwrites the buffer for nextFrame
					// this size should be set on init and done auto on getFrame - should prob be based on the canvas size type? on init?
					gocv.Resize(*nextFrame, nextFrame, image.Point{camera1.FrameBuffer.Cols(), camera1.FrameBuffer.Rows()}, 0, 0, gocv.InterpolationDefault)
				}

				fxImg := gocv.NewMat()
				defer fxImg.Close()

				// add green screen mask effect to stream frame, exposing background
				streams.AddGreenScreenMask(camera1.FrameBuffer, nextFrame, &fxImg)
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

	// close file streams
	rawWriter.Close()
	fxWriter.Close()
}
