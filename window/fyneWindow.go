package window

import (
	"fmt"
	"golang_greenscreen/streams"
	"image"
	"time"

	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/dialog"
	"gocv.io/x/gocv"
)

var StreamApp = app.New()
var StreamWindow = StreamApp.NewWindow("Stream")
var IsActive = true
var safelyQuitSignal = make(chan bool)

func init() {
	defer StreamWindow.Close()

	// set on exit dialog and cleanup
	StreamWindow.SetCloseIntercept(func() {
		confirmation := dialog.NewConfirm("Confirmation", "Are You Sure You Want to Exit?", func(response bool) {
			if response {
				IsActive = false
				<-safelyQuitSignal
				StreamApp.Quit()
			}
		}, StreamWindow)
		confirmation.Show()
	})
}

func StartCaptureStream(backgroundFeed streams.BackgroundStream, cap *streams.CaptureDevice, rawWriter gocv.VideoWriter, fxWriter gocv.VideoWriter) {
	ticker := time.NewTicker(30 * time.Millisecond)

	fyneImg, _ := streams.GetNextBackgroundBuffer(backgroundFeed).ToImage()

	fyneImage := canvas.NewImageFromImage(fyneImg)
	fyneImage.FillMode = canvas.ImageFillOriginal
	StreamWindow.SetContent(container.NewMax(fyneImage))

	for IsActive {
		select {
		case <-ticker.C:
			// handle capture device
			if !cap.NextFrame() {
				fmt.Println("Error Fetching Frame From Capture Device.")
				continue
			}

			// handle background feed
			nextBackgroundFrame := streams.GetNextBackgroundBuffer(backgroundFeed)
			if nextBackgroundFrame == nil {
				fmt.Println("Issue Getting Background Image Frame Buffer")
				continue
			}

			// resize background if needed
			if nextBackgroundFrame.Cols() != cap.FrameBuffer.Cols() || nextBackgroundFrame.Rows() != cap.FrameBuffer.Rows() {
				// TODO this overwrites the buffer for nextFrame
				// this size should be set on init and done auto on getFrame - should prob be based on the canvas size type? on init?
				gocv.Resize(*nextBackgroundFrame, nextBackgroundFrame, image.Point{cap.FrameBuffer.Cols(), cap.FrameBuffer.Rows()}, 0, 0, gocv.InterpolationDefault)
			}

			// add green screen mask effect to stream frame, exposing background
			fxImg := gocv.NewMat()
			defer fxImg.Close()

			// TODO move to main ------ write funcs
			// add green screen and save mask file
			streams.AddGreenScreenMask(cap.FrameBuffer, nextBackgroundFrame, &fxImg)
			canvasImg, _ := fxImg.ToImage()

			// record raw capture stream to file
			rawWriter.Write(*cap.FrameBuffer)

			// write fx video frame to disk
			fxWriter.Write(fxImg)
			fxImg.Close()

			// update fyne image canvas
			fyneImage.Image = canvasImg
			fyneImage.Refresh()
		}
	}
	safelyQuitSignal <- true
}
