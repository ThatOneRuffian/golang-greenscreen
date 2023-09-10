package window

import (
	"fmt"
	"golang_greenscreen/streams"
	"image"
	"log"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/widget"
	"gocv.io/x/gocv"
)

var StreamStruct *appStruct

type appStruct struct {
	StreamApp        fyne.App
	StreamWindow     fyne.Window
	IsActive         bool
	safelyQuitSignal chan bool

	// widgets
	recordBtn         *widget.Button
	captureCombSelect *widget.Select
}

func init() {
	StreamStruct = &appStruct{}
	StreamStruct.StreamApp = app.New()
	StreamStruct.StreamWindow = StreamStruct.StreamApp.NewWindow("Stream")
	defer StreamStruct.StreamWindow.Close()

	// set on exit dialog and cleanup
	StreamStruct.StreamWindow.SetCloseIntercept(func() {
		confirmation := dialog.NewConfirm("Confirmation", "Are You Sure You Want to Exit?", func(response bool) {
			if response {
				StreamStruct.IsActive = false
				<-StreamStruct.safelyQuitSignal
				StreamStruct.StreamApp.Quit()
			}
		}, StreamStruct.StreamWindow)
		confirmation.Show()
	})
}

func StartCaptureStream(backgroundFeed streams.BackgroundStream, cap *streams.CaptureDevice, rawWriter gocv.VideoWriter, fxWriter gocv.VideoWriter) {
	ticker := time.NewTicker(fpsToMilisecond(cap.FrameRate))

	fyneImg, _ := streams.GetNextBackgroundBuffer(backgroundFeed).ToImage()

	fyneImage := canvas.NewImageFromImage(fyneImg)
	fyneImage.FillMode = canvas.ImageFillOriginal

	recordBtn := widget.NewButton("Record", func() {
		fmt.Println("recording")
	})
	recordBtn.Disable()

	capCombo := widget.NewSelect(streams.AvailableCaptureDevices, func(value string) {
		log.Println("Selected set to", value)
		if value != "" {
			// init selected camera
			if cap.Connected {
				// todo this needs to point to new capture device can't close
				cap.CaptureDevice.Close()
			}
			if err := cap.InitCaptureDevice(); err != nil {
				fmt.Printf("Issue Opening Capture Device %d \n", cap.DeviceID)
			}
			recordBtn.Enable()
		}
	})

	StreamStruct.IsActive = true
	StreamStruct.recordBtn = recordBtn
	StreamStruct.captureCombSelect = capCombo
	StreamStruct.safelyQuitSignal = make(chan bool)

	// draw canvas
	StreamStruct.StreamWindow.SetContent(container.NewVBox(fyneImage, StreamStruct.captureCombSelect, StreamStruct.recordBtn))

	// fyne takes some time to setup the initial render size, so waiting here...
	go time.AfterFunc(time.Millisecond*100, func() {
		StreamStruct.StreamWindow.CenterOnScreen()
	})

	for StreamStruct.IsActive {
		select {
		case <-ticker.C:

			// handle capture device
			if !cap.NextFrame() || !cap.Connected {
				// todo handle camera not connected? default image?
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
	StreamStruct.safelyQuitSignal <- true
}

func fpsToMilisecond(fps float64) time.Duration {
	return time.Duration(1000 / fps)
}
