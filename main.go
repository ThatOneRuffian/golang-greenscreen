package main

import (
	"fmt"
	"image"

	"golang_greenscreen/streams"
	"golang_greenscreen/window"

	"gocv.io/x/gocv"
)

func init() {
	// TODO  create slice of capture devices with active/inactive status for GUI
}

func main() {

	// --------- init webcam
	camera1 := &streams.CaptureDevice{
		DeviceID:      0,
		FrameRate:     24.0,
		CaptureHeight: 480.0,
		CaptureWidth:  864.0,
	}
	if err := camera1.InitCaptureDevice(); err != nil {
		fmt.Printf("Issue Opening Capture Device %d \n", camera1.DeviceID)
	}
	defer camera1.CaptureDevice.Close()

	// --------- Load the background image
	backgroundImage := &streams.InputImage{
		SourceFile:  "./background.jpg",
		FrameBuffer: new(gocv.Mat),
	}
	defer backgroundImage.FrameBuffer.Close()
	// todo make init to load image and handle error
	// resize to canvas size on init
	*backgroundImage.FrameBuffer = gocv.IMRead(backgroundImage.SourceFile, gocv.IMReadColor)

	// resize the background image to match the frame size
	// TODO this should be updated to match the capture device on init
	gocv.Resize(*backgroundImage.FrameBuffer, backgroundImage.FrameBuffer, image.Point{int(camera1.CaptureWidth), int(camera1.CaptureHeight)}, 0, 0, gocv.InterpolationDefault)

	// ---------- Load background video background.mp4
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

	// -------------- run fyne app
	// app loop for video rendering
	go window.StartCaptureStream(backgroundVideo, camera1, *rawWriter, *fxWriter)

	window.StreamWindow.ShowAndRun()

	// close file streams
	rawWriter.Close()
	fxWriter.Close()
}
