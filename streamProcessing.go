package main

import (
	"fmt"
	"os"

	"gocv.io/x/gocv"
)

var frameStillCounter = 0
var defaultOutputDir = "./output"
var defaultImageSequenceDir = fmt.Sprintf("%s/image_sequence", defaultOutputDir)

func initOutputDir() {

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
		_, err := os.Stat(defaultImageSequenceDir)

		// if error reading dir
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Createing Output Dir: %s\n", defaultImageSequenceDir)

				if createErr := os.MkdirAll(defaultImageSequenceDir, os.FileMode(0744)); createErr != nil {
					fmt.Println(createErr)
					fmt.Println("Could Not Create Output Directory. Check Program Permissions. Cannot Save Masked Image Sequence.")
					return false
				} else if !gocv.IMWrite(fileName, resultImage) { // retry after directory creation
					fmt.Println("Unknown Issue With Writing Masked Image Sequence.")
					return false
				}
				return true
			} else if os.IsPermission(err) {
				fmt.Println("Unable to Write to Output Dir Incorrect Permissions.")
				return false
			} else {
				fmt.Println("Unknown Error Attempting to Read Output Dir, Continuing:", err)
				return false
			}
		}

		// check case of read-only access
		_, permErr := os.OpenFile(fmt.Sprintf("%s/dummyfile", defaultImageSequenceDir), os.O_RDWR, 0666)
		if permErr != nil {
			fmt.Printf("Output Directory Does Not Have the Correct Permission to Write Masked Image Sequence.\n")
		}
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
