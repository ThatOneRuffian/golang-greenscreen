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
var defaultOutputDir = "./output"

type appStruct struct {
	StreamApp    fyne.App
	StreamWindow fyne.Window

	// signals
	safelyQuitSignal chan bool

	// states
	streamIsActive    bool
	streamIsRecording bool

	// widgets
	recordBtn         *widget.Button
	captureCombSelect *widget.Select
}

func init() {
	StreamStruct = &appStruct{}
	StreamStruct.StreamApp = app.New()
	StreamStruct.StreamWindow = StreamStruct.StreamApp.NewWindow("Stream")

	// set on exit dialog and cleanup
	StreamStruct.StreamWindow.SetCloseIntercept(func() {
		confirmation := dialog.NewConfirm("Confirmation", "Are You Sure You Want to Exit?", func(response bool) {
			if response {

				if StreamStruct.streamIsActive {
					StreamStruct.streamIsActive = false
					<-StreamStruct.safelyQuitSignal
				}

				StreamStruct.StreamApp.Quit()
			}
		}, StreamStruct.StreamWindow)
		confirmation.Show()
	})
}

func StartMainWindow(backgroundFeed streams.BackgroundStream) {

	// --------- init webcam
	cap := &streams.CaptureDevice{
		//DeviceID: 0,
		//FrameRate: 24.0,
		//CaptureHeight: 480,
		//CaptureWidth:  864,
	}

	streamWriters := &streams.WriterPipeLine{}
	fyneImg, bgErr := streams.GetNextBackgroundBuffer(backgroundFeed).ToImage()

	if bgErr != nil {
		fmt.Println("Could Not Aquire Next Frame From Background Stream. Using Default.")
		//TODO set default background iamge
	}

	recordStopSig := make(chan bool, 2)
	// calculate the scaling factor based on the desired maxWidth and the original width
	// width for canvas
	maxWidth := 200.0
	scaleFactor := maxWidth / float64(fyneImg.Bounds().Dx())
	fyneImage := canvas.NewImageFromImage(fyneImg)
	canvasHeight := int(float64(fyneImg.Bounds().Dy()) * scaleFactor)
	fyneImage.Resize(fyne.NewSize(int(maxWidth), canvasHeight))
	fyneImage.FillMode = canvas.ImageFillContain

	recordBtn := widget.NewButton("Record", func() {
		// TODO need to track recording status

		if !StreamStruct.streamIsRecording {
			fmt.Println("recording")
			// --------- init media output dir
			// TODO
			// need to add timestamp to output dir to avoid overwrites - need structure: output > day_start > session_id > record_hash_id_count? > [img, mp4, mp4]
			// fix bug where dir is not found and insta crash on write "./output2" gocv panic
			currentRecordDir := fmt.Sprintf("%s/%s", defaultOutputDir, time.Now().Format("2006-01-02-150405"))
			defaultImageSequenceDir := fmt.Sprintf("%s/image_sequence", currentRecordDir)

			if err := streams.InitOutputDir(defaultImageSequenceDir); err != nil {
				fmt.Printf("There Was an Error Creating the Stream Output Directory: %v", err)
				StreamStruct.streamIsRecording = false
			}

			// todo update button to reflect state
			rawErr, fxErr := streamWriters.OpenWriters(currentRecordDir, cap)

			if rawErr != nil {
				fmt.Println("Error Opening Raw Writer.")
			}

			if fxErr != nil {
				fmt.Println("Error Opening FX Writer.")
			}

			// begin recording stream
			StreamStruct.streamIsRecording = true

		} else {
			// already recording close recorders
			//todo revert back to normal button

			// todo need to wait for frames to be written button is in thread
			fmt.Println("recording stopped")
			recordStopSig <- true
		}
	})
	recordBtn.Disable()

	capCombo := widget.NewSelect(streams.AvailableCaptureDevices, func(value string) {
		log.Println("Selected Capture Device Set to", value)
		if value != "" {
			recordStopSig <- true
			StreamStruct.streamIsActive = false

			// init selected camera
			if cap.Connected {
				// todo this needs to point to new capture device can't close
				cap.CaptureDevice.Close()
			}
			if err := cap.InitCaptureDevice(value); err != nil {
				fmt.Printf("Issue Opening Capture Device %s \n", value)
			}
			StreamStruct.streamIsActive = true
			go startCaptureStream(cap, backgroundFeed, streamWriters, fyneImage, recordStopSig)
			recordBtn.Enable()
		}
	})

	StreamStruct.recordBtn = recordBtn
	StreamStruct.captureCombSelect = capCombo
	StreamStruct.safelyQuitSignal = make(chan bool)

	// draw canvas
	StreamStruct.StreamWindow.SetContent(container.NewAdaptiveGrid(1, fyneImage, StreamStruct.captureCombSelect, StreamStruct.recordBtn))
}

func startCaptureStream(cap *streams.CaptureDevice, backgroundFeed streams.BackgroundStream, streamWriters *streams.WriterPipeLine, fyneImage *canvas.Image, recordStopSig chan bool) {
	maxWidth := 200.0
	ticker := time.NewTicker(fpsToMilisecond(cap.FrameRate))

	for StreamStruct.streamIsActive {
		select {
		case <-ticker.C:
			// need to update tick to match camera's fps
			if len(recordStopSig) > 0 {
				fmt.Println("Record Stop Signal Received")
				StreamStruct.streamIsRecording = false
				streamWriters.MaskedStillCounter = 0
				streamWriters.CloseWriters()
				<-recordStopSig
			}

			// handle capture device
			if !cap.NextFrame() || !cap.Connected {
				// todo handle camera not connected? default image and scaling of image?
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

			// TODO add fx pipeline
			// add green screen effect and save mask file
			streams.AddGreenScreenMask(cap.FrameBuffer, nextBackgroundFrame, &fxImg)
			canvasImg, _ := fxImg.ToImage()

			// Calculate the scaling factor based on the desired maxWidth and the original width
			scaleFactor := maxWidth / float64(canvasImg.Bounds().Dx())

			// save images to writer pipeline
			if StreamStruct.streamIsRecording {
				var rawErr, fxErr, maskErr error
				rawErr, fxErr, maskErr = streamWriters.SaveFrames(cap.FrameBuffer, &fxImg, &fxImg)
				// todo handle these errors need stderr and debug
				_ = rawErr
				_ = fxErr
				_ = maskErr
				//fmt.Println(rawErr, fxErr, maskErr)
			}
			fxImg.Close()

			// update fyne image canvas
			fyneImage.Image = canvasImg
			fyneImage.Resize(fyne.NewSize(int(maxWidth), int(float64(canvasImg.Bounds().Dy())*scaleFactor)))
			fyneImage.Refresh()
			StreamStruct.StreamWindow.Content().Refresh()
		}
	}
	streamWriters.CloseWriters()
	StreamStruct.safelyQuitSignal <- true
}

func fpsToMilisecond(fps float64) time.Duration {
	return time.Duration(1000 / fps)
}
