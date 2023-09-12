package streams

import (
	"errors"
	"fmt"
	"image"
	"os"

	"gocv.io/x/gocv"
)

var AvailableCaptureDevices []string

func init() {
	AvailableCaptureDevices = enumerateCaptureDevices()
}

var canvasSize = 0

// Define the lower and upper bounds chroma key for the green color in HSV
var lowerGreen = gocv.NewScalar(22, 6, 35, 0) // greenish
var upperGreen = gocv.NewScalar(85, 255, 255, 0)

type BackgroundStream interface {
	getFrame() *gocv.Mat
}

type WriterPipeLine struct {
	MaskedImageWriteLoc string
	MaskedStillCounter  uint64
	FxStreamWriter      *gocv.VideoWriter
	RawStreamWriter     *gocv.VideoWriter
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

func (writers *WriterPipeLine) SaveFrames(rawFrame *gocv.Mat, maskedFrame *gocv.Mat, fxFrame *gocv.Mat) (error, error, error) {
	// TODO this should probably be async
	var maskErr error

	// record raw capture frame to file
	rawErr := writers.RawStreamWriter.Write(*rawFrame)

	// write fx video frame to disk
	fxErr := writers.FxStreamWriter.Write(*fxFrame)

	// write masked image to disk
	maskFileName := fmt.Sprintf("%s/output_image_%d.png", writers.MaskedImageWriteLoc, writers.MaskedStillCounter)

	if maskSaved := gocv.IMWrite(maskFileName, *maskedFrame); maskSaved {
		writers.MaskedStillCounter += 1
		maskErr = nil
	}

	return rawErr, fxErr, maskErr
}

func (writers *WriterPipeLine) OpenWriters(dir string) (error, error) {

	frameRate := 24.0
	frameWidth := 864
	frameHeight := 480

	// create writer for VFX stream
	fxSaveFile := fmt.Sprintf("%s/stream_fx_output.mp4", dir)
	var fxErr error
	writers.FxStreamWriter, fxErr = gocv.VideoWriterFile(fxSaveFile, "mp4v", frameRate, frameWidth, frameHeight, true)
	if fxErr != nil {
		fmt.Printf("Error Opening FX Video Writer: %v\n", fxSaveFile)
		fmt.Printf("err: %v\n", fxErr)
	}

	// create writer for raw stream
	rawSaveFile := fmt.Sprintf("%s/stream_raw_output.mp4", dir)
	var rawErr error
	writers.RawStreamWriter, rawErr = gocv.VideoWriterFile(rawSaveFile, "mp4v", frameRate, frameWidth, frameHeight, true)
	if rawErr != nil {
		fmt.Printf("Error Opening Raw Video Writer: %v\n", rawSaveFile)
		fmt.Printf("err: %v\n", rawErr)
	}

	// set mask save dir
	writers.MaskedImageWriteLoc = fmt.Sprintf("%s/image_sequence/", dir)

	return rawErr, fxErr
}

func (writers *WriterPipeLine) CloseWriters() {
	writers.FxStreamWriter.Close()
	writers.RawStreamWriter.Close()
}

func saveFrameWithMaskAlpha(sourceImage *gocv.Mat, mask *gocv.Mat) *gocv.Mat {
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
	gocv.Merge(channels, &resultImage)

	return &resultImage
}

func AddGreenScreenMask(sourceImage *gocv.Mat, newBackground *gocv.Mat, result *gocv.Mat) {

	// Convert sourceImage to the HSV color space for chroma keying
	hsvImg := gocv.NewMat()
	defer hsvImg.Close()
	gocv.CvtColor(*sourceImage, &hsvImg, gocv.ColorBGRToHSV)

	// Create a mask by thresholding the image within the specified HSV range
	mask := gocv.NewMat()
	defer mask.Close()
	gocv.InRangeWithScalar(hsvImg, lowerGreen, upperGreen, &mask)

	// Invert mask for applying on the original image
	invertedMask := gocv.NewMat()
	defer invertedMask.Close()
	gocv.BitwiseNot(mask, &invertedMask)

	// Mask the source image and the new background with the respective masks
	sourceMasked := gocv.NewMat()
	defer sourceMasked.Close()
	gocv.BitwiseAndWithMask(*sourceImage, *sourceImage, &sourceMasked, invertedMask)

	backgroundMasked := gocv.NewMat()
	defer backgroundMasked.Close()
	newBackground.CopyToWithMask(&backgroundMasked, mask)

	// Combine the masked source image and the masked background for final fx
	gocv.Add(sourceMasked, backgroundMasked, result)
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

func InitOutputDir(saveDir string) error {
	// check if output dir exists
	_, err := os.Stat(saveDir)

	if err != nil {
		// test read/execute permissions
		switch {
		case os.IsNotExist(err): // output dir does not exist
			if createErr := os.MkdirAll(saveDir, os.FileMode(0744)); createErr != nil {
				return errors.New("Could Not Create Output Directory. Check Program Permissions. Cannot Save Masked Image Sequence.")

			}
			fmt.Printf("Created Output Dir: %s\n", saveDir)
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
	dummyFile := fmt.Sprintf("%v/dummyfile.tmp", saveDir)
	fileInfo, permErr := os.Create(dummyFile)
	fileInfo.Close()

	if remErr := os.Remove(dummyFile); remErr != nil {
		fmt.Println("Error Removing Dummy File. Unknown Error.")
	}

	if permErr != nil {
		return errors.New(fmt.Sprintf("Output Directory Does Not Have Required Write Permissions. Unable to Write Output Media.\n%v", permErr))
	}

	return nil
}
