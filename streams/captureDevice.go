package streams

import (
	"fmt"

	"gocv.io/x/gocv"
)

type CaptureDevice struct {
	DeviceID      int
	Connected     bool
	FrameRate     float64
	CaptureHeight float64
	CaptureWidth  float64
	CaptureDevice *gocv.VideoCapture
	FrameBuffer   *gocv.Mat
}

// todo need to take camera and settings
// return camera instance with pointers to current settings
// or something for UI. need dropdown menu select feed(s)
func (cap *CaptureDevice) InitCaptureDevice() error {
	fmt.Printf("Attempting to mount capture device %d...", cap.DeviceID)
	var err error
	cap.CaptureDevice, err = gocv.VideoCaptureDeviceWithAPI(cap.DeviceID, gocv.VideoCaptureGstreamer)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", cap.DeviceID)
		fmt.Println(err)
		return err
	}

	// set camera's capture settings
	cap.CaptureDevice.Set(gocv.VideoCaptureFPS, cap.FrameRate)
	cap.CaptureDevice.Set(gocv.VideoCaptureFrameHeight, cap.CaptureHeight)
	cap.CaptureDevice.Set(gocv.VideoCaptureFrameWidth, cap.CaptureWidth)

	// init frame buffer
	img := gocv.NewMat()
	cap.FrameBuffer = &img

	// print camera's current settings
	width := cap.CaptureDevice.Get(gocv.VideoCaptureFrameWidth)
	height := cap.CaptureDevice.Get(gocv.VideoCaptureFrameHeight)
	FrameRate := cap.CaptureDevice.Get(gocv.VideoCaptureFPS)
	fmt.Println(width, height, FrameRate)
	cap.Connected = true

	return nil
}

func (cap *CaptureDevice) NextFrame() bool {
	// read in the next frame into the capture device's frame buffer
	if cap.Connected && cap.FrameBuffer != nil {
		if ok := cap.CaptureDevice.Read(cap.FrameBuffer); !ok {
			fmt.Printf("Device closed: %v\n", cap.DeviceID)
			return false
		}
		if cap.FrameBuffer.Empty() {
			fmt.Println("Capture Device Returned Empty Frame Buffer.")
			return false
		}
		return true
	}
	return false
}
