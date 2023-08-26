package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

var Window = gocv.NewWindow("Feed Preview")
var frameCount = 0

func main() {
	defer Window.Close()
	// --------- init webcam
	deviceID := 0
	targetFrameRate := 24.0
	targetFrameHeight := 480.0
	targetFframeWidth := 864.0

	webcam, err := gocv.VideoCaptureDeviceWithAPI(deviceID, gocv.VideoCaptureGstreamer)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", deviceID)
		fmt.Println(err)
		return
	}
	defer webcam.Close()

	// set camera's capture settings
	webcam.Set(gocv.VideoCaptureFPS, targetFrameRate)
	webcam.Set(gocv.VideoCaptureFrameHeight, targetFrameHeight)
	webcam.Set(gocv.VideoCaptureFrameWidth, targetFframeWidth)

	// print camera's actual settings
	width := float64(webcam.Get(gocv.VideoCaptureFrameWidth))
	height := float64(webcam.Get(gocv.VideoCaptureFrameHeight))
	frameRate := float64(webcam.Get(gocv.VideoCaptureFPS))
	fmt.Println(width, height, frameRate)

	// create image variable to hold camera frames
	img := gocv.NewMat()
	defer img.Close()
	fmt.Printf("Start reading device: %v\n", deviceID)
	if ok := webcam.Read(&img); !ok {
		fmt.Printf("Device closed: %v\n", deviceID)
		return
	}
	fmt.Println(img.Cols(), img.Rows())

	// Load the background image
	backgroundPath := "./background.jpg" // Specify the path to your background image
	background := gocv.IMRead(backgroundPath, gocv.IMReadColor)
	defer background.Close()

	// Load background video background.mkv
	backgroundVideoPath := "./background.mp4" // Specify the path to your background video
	backgroundVideo, err := gocv.VideoCaptureFile(backgroundVideoPath)
	if err != nil {
		fmt.Printf("Error opening video file: %v\n", err)
		return
	}
	defer backgroundVideo.Close()

	videoFrame := gocv.NewMat()
	defer videoFrame.Close()

	// Resize the background image to match the frame size
	gocv.Resize(background, &background, image.Point{img.Cols(), img.Rows()}, 0, 0, gocv.InterpolationDefault)

	// create writer for raw stream
	rawSaveFile := "stream_raw_output.mp4"
	rawWriter, err := gocv.VideoWriterFile(rawSaveFile, "mp4v", targetFrameRate, img.Cols(), img.Rows(), true)
	if err != nil {
		fmt.Printf("error opening video writer device: %v\n", rawSaveFile)
		fmt.Printf("err: %v\n", err)
		return
	}
	defer rawWriter.Close()

	// create writer for VFX stream
	fxSaveFile := "stream_fx_output.mp4"
	fxWriter, err := gocv.VideoWriterFile(fxSaveFile, "mp4v", targetFrameRate, img.Cols(), img.Rows(), true)
	if err != nil {
		fmt.Printf("error opening video writer device: %v\n", fxSaveFile)
		fmt.Printf("err: %v\n", err)
		return
	}
	defer fxWriter.Close()

	for {
		// capture next video frame from webcam
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		// capture next video frame from file
		if ok := backgroundVideo.Read(&videoFrame); !ok {
			break
		}

		if videoFrame.Empty() {
			break
		}

		gocv.Resize(videoFrame, &videoFrame, image.Point{img.Cols(), img.Rows()}, 0, 0, gocv.InterpolationDefault)

		// record raw stream to file
		rawWriter.Write(img)

		// todo need to save webm transparency video file?

		// add green screen mask effect to stream frame, exposing background
		fxImg := gocv.NewMat()
		defer fxImg.Close()

		addGreenScreenMask(&img, &videoFrame, &fxImg)

		// write fx video frame
		fxWriter.Write(fxImg)
		fxImg.Close()

		// ESC to shutdown program
		if Window.WaitKey(1) == 27 {
			break
		}
	}
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
	saveFrameWithAlpha(sourceImage, &invertedMask)

	// Create a result image by bitwise-AND between the original image and the mask
	gocv.BitwiseAndWithMask(*sourceImage, *sourceImage, result, invertedMask)

	// Add the masked frame and background
	gocv.Add(*result, backgroundResult, result)

	// Update window
	Window.IMShow(*result)
}

func saveFrameWithAlpha(sourceImage *gocv.Mat, mask *gocv.Mat) bool {
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

	// Save the transparent image as a PNG file
	fileName := fmt.Sprintf("./output/output_image_%d.png", frameCount)
	frameCount += 1

	return gocv.IMWrite(fileName, resultImage)
}
