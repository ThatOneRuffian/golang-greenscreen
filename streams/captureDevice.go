package streams

import (
	"fmt"
	"strconv"

	"gocv.io/x/gocv"
)

var AvailableCaptureDevices []string

func init() {
	AvailableCaptureDevices = enumerateCaptureDevices()
}

type CaptureDevice struct {
	DeviceID      int
	Connected     bool
	FrameRate     float64
	CaptureHeight int
	CaptureWidth  int
	CaptureDevice *gocv.VideoCapture
	FrameBuffer   *gocv.Mat
}

// todo need to take camera and settings
// return camera instance with pointers to current settings
// or something for UI. need dropdown menu select feed(s)
func (cap *CaptureDevice) InitCaptureDevice(selectedCaptureDevice string) error {
	fmt.Printf("Attempting to mount capture device %s...", selectedCaptureDevice)
	var capErr error
	var convErr error
	decviceId, convErr := strconv.Atoi(selectedCaptureDevice)
	cap.CaptureDevice, capErr = gocv.VideoCaptureDeviceWithAPI(decviceId, gocv.VideoCaptureV4L2)
	if capErr != nil || convErr != nil {
		fmt.Printf("Error opening video capture device %s:\n", selectedCaptureDevice)
		fmt.Println(capErr)
		return capErr
	}
	cap.Connected = true
	cap.DeviceID = decviceId

	// set camera's capture settings
	//cap.CaptureDevice.Set(gocv.VideoCaptureFPS, cap.FrameRate)
	//cap.CaptureDevice.Set(gocv.VideoCaptureFrameHeight, float64(cap.CaptureHeight))
	//cap.CaptureDevice.Set(gocv.VideoCaptureFrameWidth, float64(cap.CaptureWidth))

	// init frame buffer
	img := gocv.NewMat()
	cap.FrameBuffer = &img

	// init camera's current settings
	cap.CaptureWidth = int(cap.CaptureDevice.Get(gocv.VideoCaptureFrameWidth))
	cap.CaptureHeight = int(cap.CaptureDevice.Get(gocv.VideoCaptureFrameHeight))
	cap.FrameRate = float64(cap.CaptureDevice.Get(gocv.VideoCaptureFPS))
	fmt.Println(cap.CaptureWidth, cap.CaptureHeight, cap.FrameRate)

	return nil
}

func (cap *CaptureDevice) NextFrame() bool {
	// read in the next frame into the capture device's frame buffer
	if cap.Connected && cap.CaptureDevice != nil && cap.FrameBuffer != nil {
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

func enumerateCaptureDevices() []string {
	var discoveredDevices []string
	for i := 0; i < 10; i++ {
		webcam, err := gocv.OpenVideoCapture(i)
		if err != nil {
			fmt.Printf("Error Opening Capture Device %d During Enumeration: %s\n", i, err)
			continue
		} else if !webcam.IsOpened() {
			fmt.Println("Device in use. Need to update GUI on this")
			continue
		}
		discoveredDevices = append(discoveredDevices, fmt.Sprintf("%d", i))
		fmt.Printf("Found capture device at index %d\n", i)
		webcam.Close()
	}
	return discoveredDevices
}
