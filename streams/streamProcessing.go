package streams

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

var AvailableCaptureDevices []string

func init() {
	AvailableCaptureDevices = enumerateCaptureDevices()
}

var canvasSize = 0

type BackgroundStream interface {
	getFrame() *gocv.Mat
}

type WriterPipeLine struct {
	FxStreamWriter    *gocv.VideoWriter
	RawStreamWriter   *gocv.VideoWriter
	MaskedImageWriter func(gocv.Mat) error
}

type InputVideo struct {
	SourceFile  string
	FrameBuffer *gocv.Mat
	VideoReader *gocv.VideoCapture
	FrameSize   image.Point
}

type InputImage struct {
	SourceFile  string
	FrameBuffer *gocv.Mat
	FrameSize   image.Point
}

func GetNextBackgroundBuffer(backgroundFeed BackgroundStream) *gocv.Mat {
	return backgroundFeed.getFrame()
}

func (InputVideo *InputVideo) getFrame() *gocv.Mat {
	// capture next video frame from file
	if ok := InputVideo.VideoReader.Read(InputVideo.FrameBuffer); !ok {
		// attempt to set video file to first frame and reread
		// for EOF condition
		InputVideo.VideoReader.Set(gocv.VideoCapturePosFrames, 0)
		if ok := InputVideo.VideoReader.Read(InputVideo.FrameBuffer); !ok {
			return nil
		}
	}
	if InputVideo.FrameBuffer.Empty() {
		fmt.Println("Empty Frame Buffer Received From Capture Device.")
		return nil
	}
	return InputVideo.FrameBuffer
}

func (img *InputImage) getFrame() *gocv.Mat {
	return img.FrameBuffer
}

func (img *InputImage) resizeFrame() *gocv.Mat {
	return img.FrameBuffer
}

func saveFrameWithMaskAlpha(sourceImage *gocv.Mat, mask *gocv.Mat) bool {
	// Ensure mask has only 0 and 255 values
	gocv.Threshold(*mask, mask, 128, 255, gocv.ThresholdBinary)

	// Create a new image with an alpha channel (BGRA)
	rgbaImage := gocv.NewMat()
	defer rgbaImage.Close()
	gocv.CvtColor(*sourceImage, &rgbaImage, gocv.ColorBGRToBGRA)

	// Create the alpha channel from the mask
	alphaChannel := gocv.NewMatWithSize(rgbaImage.Rows(), rgbaImage.Cols(), gocv.MatTypeCV8U)
	defer alphaChannel.Close()

	// Set the alpha channel values based on the mask
	mask.ConvertTo(&alphaChannel, gocv.MatTypeCV8U)

	// Split the BGRA image into its components
	channels := gocv.Split(rgbaImage)

	// Set the alpha channel in the BGRA image
	defer channels[0].Close()
	defer channels[1].Close()
	defer channels[2].Close()
	channels[3].Close() // Close the original alpha channel from rgbaImage
	channels[3] = alphaChannel

	// Merge the BGRA components back into the final image
	resultImage := gocv.NewMat()
	defer resultImage.Close()
	gocv.Merge(channels, &resultImage)

	fileName := fmt.Sprintf("%s/output_image_%d.png", defaultImageSequenceDir, frameStillCounter)
	frameStillCounter += 1
	// Save the transparent image as a PNG file
	// and attempt to determine issues with write
	if !gocv.IMWrite(fileName, resultImage) {
		fmt.Println("Unknown Issue While Saving Masked Frames to Output Dir.")
	}
	return true
}

func AddGreenScreenMask(sourceImage *gocv.Mat, newBackground *gocv.Mat, result *gocv.Mat) {
	// create capture window

	// Define the lower and upper bounds for the green color in HSV
	lowerGreen := gocv.NewScalar(22, 6, 35, 0)
	upperGreen := gocv.NewScalar(85, 255, 255, 0)

	// Convert the image to the HSV color space for green screening
	hsvImg := gocv.NewMat()
	defer hsvImg.Close()
	gocv.CvtColor(*sourceImage, &hsvImg, gocv.ColorBGRToHSV)

	// Create a mask by thresholding the image within the specified HSV range
	mask := gocv.NewMat()
	defer mask.Close()
	gocv.InRangeWithScalar(hsvImg, lowerGreen, upperGreen, &mask)

	// Apply the mask to the background to drop green out
	backgroundResult := gocv.NewMat()
	defer backgroundResult.Close()
	newBackground.CopyToWithMask(&backgroundResult, mask)

	// Invert mask for the displaying of the background image
	invertedMask := gocv.NewMat()
	defer invertedMask.Close()

	gocv.BitwiseNot(mask, &invertedMask)
	saveFrameWithMaskAlpha(sourceImage, &invertedMask)

	// Create a result image by bitwise-AND between the original image and the mask
	gocv.BitwiseAndWithMask(*sourceImage, *sourceImage, result, invertedMask)

	// Add the masked frame and background
	gocv.Add(*result, backgroundResult, result)
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
