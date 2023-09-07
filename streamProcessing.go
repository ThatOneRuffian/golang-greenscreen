package main

import (
	"errors"
	"fmt"
	"os"

	"gocv.io/x/gocv"
)

var frameStillCounter = 0
var defaultOutputDir = "./output"
var defaultImageSequenceDir = fmt.Sprintf("%s/image_sequence", defaultOutputDir)

func initOutputDir() error {
	// check if output dir exists
	_, err := os.Stat(defaultImageSequenceDir)

	if err != nil {
		// test read/execute permissions
		switch {
		case os.IsNotExist(err): // output dir does not exist
			fmt.Printf("Createing Output Dir: %s\n", defaultImageSequenceDir)
			if createErr := os.MkdirAll(defaultImageSequenceDir, os.FileMode(0744)); createErr != nil {
				return errors.New("Could Not Create Output Directory. Check Program Permissions. Cannot Save Masked Image Sequence.")

			}
			return nil
		case os.IsPermission(err): // no read access
			fmt.Println("Unable to Read Output Dir Incorrect Permissions.")
			fmt.Println(err)
			return errors.New("Could Not Create Output Directory. Check Program Permissions. Cannot Save Masked Image Sequence.")

		default:
			return errors.New(fmt.Sprintf("Unknown Error Attempting to Read Output Dir, Continuing: %v", err))
		}
	}

	// test write permission
	fileInfo, permErr := os.Create(fmt.Sprintf("%v/dummyfile.tmp", defaultImageSequenceDir))
	fileInfo.Close()

	if permErr != nil {
		return errors.New(fmt.Sprintf("Output Directory Does Not Have Required Write Permissions. Unable to Write Output Media.\n%v", permErr))
	}

	return nil
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
