package main

import (
	"fmt"

	"golang_greenscreen/streams"
	"golang_greenscreen/window"

	"gocv.io/x/gocv"
)

func init() {
	// TODO  create slice of capture devices with active/inactive status for GUI
}

func main() {

	// --------- Load the background image
	backgroundImage := &streams.InputImage{
		SourceFile:  "./background.jpg",
		FrameBuffer: new(gocv.Mat),
	}
	defer backgroundImage.FrameBuffer.Close()
	// todo make init to load image and handle error
	// resize to canvas size on init
	*backgroundImage.FrameBuffer = gocv.IMRead(backgroundImage.SourceFile, gocv.IMReadColor)

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

	// -------------- run fyne app
	// app loop for video rendering
	window.StartMainWindow(backgroundVideo)

	window.StreamStruct.StreamWindow.ShowAndRun()
}
