package streams

import (
	"errors"
	"fmt"
	"os"
)

func init() {
	// --------- init media output dir
	if err := InitOutputDir(); err != nil {
		panic(err)
	}
}

var frameStillCounter = 0
var DefaultOutputDir = "./output"
var defaultImageSequenceDir = fmt.Sprintf("%s/image_sequence", DefaultOutputDir)

func InitOutputDir() error {
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
