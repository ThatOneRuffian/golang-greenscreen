package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

var Window = gocv.NewWindow("Feed Preview")
var canvasSize = 0

type backgroundStream interface {
	getFrame() *gocv.Mat
}

type inputVideo struct {
	sourceFile  string
	frameBuffer *gocv.Mat
	videoReader *gocv.VideoCapture
	frameSize   image.Point
}

type inputImage struct {
	sourceFile  string
	frameBuffer *gocv.Mat
	frameSize   image.Point
}

func getBackgroundBuffer(backgroundFeed backgroundStream) *gocv.Mat {
	return backgroundFeed.getFrame()
}

func (inputVideo *inputVideo) getFrame() *gocv.Mat {
	// capture next video frame from file
	if ok := inputVideo.videoReader.Read(inputVideo.frameBuffer); !ok {
		// attempt to set video file to first frame and reread
		// for EOF condition
		inputVideo.videoReader.Set(gocv.VideoCapturePosFrames, 0)
		if ok := inputVideo.videoReader.Read(inputVideo.frameBuffer); !ok {
			return nil
		}
	}
	if inputVideo.frameBuffer.Empty() {
		fmt.Println("Empty Frame Buffer Received From Capture Device.")
		return nil
	}
	return inputVideo.frameBuffer
}

func (img *inputImage) getFrame() *gocv.Mat {
	return img.frameBuffer
}

func (img *inputImage) resizeFrame() *gocv.Mat {
	return img.frameBuffer
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

func addGreenScreenMask(sourceImage *gocv.Mat, newBackground *gocv.Mat, result *gocv.Mat) {
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
