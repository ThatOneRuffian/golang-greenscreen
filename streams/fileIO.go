package streams

import (
	"errors"
	"fmt"
	"os"

	"gocv.io/x/gocv"
)

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

func (writers *WriterPipeLine) OpenWriters(dir string, cap *CaptureDevice) (error, error) {

	// create writer for VFX stream
	fxSaveFile := fmt.Sprintf("%s/stream_fx_output.mp4", dir)
	var fxErr error
	writers.FxStreamWriter, fxErr = gocv.VideoWriterFile(fxSaveFile, "mp4v", cap.FrameRate, cap.FrameBuffer.Cols(), cap.FrameBuffer.Rows(), true)
	if fxErr != nil {
		fmt.Printf("Error Opening FX Video Writer: %v\n", fxSaveFile)
		fmt.Printf("err: %v\n", fxErr)
	}

	// create writer for raw stream
	rawSaveFile := fmt.Sprintf("%s/stream_raw_output.mp4", dir)
	var rawErr error
	writers.RawStreamWriter, rawErr = gocv.VideoWriterFile(rawSaveFile, "mp4v", cap.FrameRate, cap.FrameBuffer.Cols(), cap.FrameBuffer.Rows(), true)
	if rawErr != nil {
		fmt.Printf("Error Opening Raw Video Writer: %v\n", rawSaveFile)
		fmt.Printf("err: %v\n", rawErr)
	}

	// set mask save dir
	writers.MaskedImageWriteLoc = fmt.Sprintf("%s/image_sequence/", dir) // resize the background image to match the frame size

	return rawErr, fxErr
}

func (writers *WriterPipeLine) CloseWriters() {
	if writers.FxStreamWriter != nil {
		writers.FxStreamWriter.Close()
	}
	if writers.RawStreamWriter != nil {
		writers.RawStreamWriter.Close()
	}
}
