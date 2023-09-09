package streams

import (
	"fmt"

	"gocv.io/x/gocv"
)

type captureDevice struct {
	deviceID      int
	connected     bool
	frameRate     float64
	captureHeight float64
	captureWidth  float64
	captureDevice *gocv.VideoCapture
	frameBuffer   *gocv.Mat
}

// todo need to take camera and settings
// return camera instance with pointers to current settings
// or something for UI. need dropdown menu select feed(s)
func (cap *captureDevice) initCaptureDevice() error {
	fmt.Printf("Attempting to mount capture device %d...", cap.deviceID)
	var err error
	cap.captureDevice, err = gocv.VideoCaptureDeviceWithAPI(cap.deviceID, gocv.VideoCaptureGstreamer)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", cap.deviceID)
		fmt.Println(err)
		return err
	}

	// set camera's capture settings
	cap.captureDevice.Set(gocv.VideoCaptureFPS, cap.frameRate)
	cap.captureDevice.Set(gocv.VideoCaptureFrameHeight, cap.captureHeight)
	cap.captureDevice.Set(gocv.VideoCaptureFrameWidth, cap.captureWidth)

	// init frame buffer
	img := gocv.NewMat()
	cap.frameBuffer = &img

	// print camera's current settings
	width := cap.captureDevice.Get(gocv.VideoCaptureFrameWidth)
	height := cap.captureDevice.Get(gocv.VideoCaptureFrameHeight)
	frameRate := cap.captureDevice.Get(gocv.VideoCaptureFPS)
	fmt.Println(width, height, frameRate)
	cap.connected = true

	return nil
}

func (cap *captureDevice) nextFrame() bool {
	// read in the next frame into the capture device's frame buffer
	if cap.connected && cap.frameBuffer != nil {
		if ok := cap.captureDevice.Read(cap.frameBuffer); !ok {
			fmt.Printf("Device closed: %v\n", cap.deviceID)
			return false
		}
		if cap.frameBuffer.Empty() {
			fmt.Println("Capture Device Returned Empty Frame Buffer.")
			return false
		}
		return true
	}
	return false
}
