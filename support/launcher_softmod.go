package support

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
)

type zipFilesData struct {
	Name string
	Data []byte
}

/* Used for reading softmod directory */
func readFolder(path string, sdir string) []zipFilesData {

	var zipFiles []zipFilesData

	/* Get all softmod files */
	sFiles, err := os.ReadDir(path)
	if err != nil {
		cwlog.DoLogCW("Unable to read softmod folder.")
		return nil
	}

	for _, file := range sFiles {
		if !file.IsDir() {
			dat, err := os.ReadFile(path + "/" + file.Name())
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to read softmod files.")
				continue
			}

			zipFiles = append(zipFiles, zipFilesData{Name: sdir + "/" + file.Name(), Data: dat})
		} else {
			tfiles := readFolder(path+"/"+file.Name(), sdir+"/"+file.Name())
			zipFiles = append(zipFiles, tfiles...)
		}
	}

	return zipFiles
}

/* Insert our softmod files into the save zip */
func injectSoftMod(fileName, folderName string) {
	var zipFiles []zipFilesData

	/* Read needed files from existing save */
	archive, errz := zip.OpenReader(fileName)
	if errz != nil {
		cwlog.DoLogCW("sm-inject: unable to open save game.")
		return
	} else {
		defer archive.Close()
		for _, f := range archive.File {
			fileName := path.Base(f.Name)
			/* Make sure these files are in the correct directory in the zip */
			if strings.Compare(path.Dir(f.Name), folderName) == 0 &&
				/* Only copy relevant files */
				strings.HasPrefix(fileName, "level.dat") ||
				strings.HasSuffix(fileName, ".json") ||
				strings.HasSuffix(fileName, ".dat") ||
				strings.EqualFold(fileName, "level-init.dat") ||
				strings.EqualFold(fileName, "level.datmetadata") {
				file, err := f.Open()
				if err != nil {
					cwlog.DoLogCW("sm-inject: unable to open " + f.Name)
				} else {

					defer file.Close()
					data, rerr := io.ReadAll(file)

					dlen := uint64(len(data))
					if rerr != nil && rerr != io.EOF {
						cwlog.DoLogCW("Unable to read file: " + f.Name)
						continue
					} else if dlen != f.UncompressedSize64 {
						sbuf := fmt.Sprintf("%v vs %v", dlen, f.UncompressedSize64)
						cwlog.DoLogCW("Sizes did not match: " + f.Name + ", " + sbuf)
					} else {
						tmp := zipFilesData{Name: f.Name, Data: data}
						zipFiles = append(zipFiles, tmp)
					}
				}
			}
		}

		/* Read files in from softmod */
		blackList := []string{"img-source", "out"}                   /* Wildcard exclude */
		allowList := []string{"README.md", "preview.jpg", "LICENSE"} /* Always include */
		allowExt := []string{".lua", ".png", ".cfg"}

		tfiles := readFolder(cfg.Local.Options.SoftModOptions.SoftModPath, folderName)
		var addFiles []zipFilesData
		for _, tf := range tfiles {
			skip := false
			for _, al := range allowList {
				if strings.HasSuffix(tf.Name, al) {
					addFiles = append(addFiles, tf)
				}
			}
			for _, bl := range blackList {
				if strings.Contains(tf.Name, bl) {
					skip = true
				}
			}
			if skip {
				continue
			}
			for _, ext := range allowExt {
				if strings.HasSuffix(tf.Name, ext) {
					addFiles = append(addFiles, tf)
				}
			}
		}
		zipFiles = append(zipFiles, addFiles...)

		numFiles := len(zipFiles)
		if numFiles <= 0 {
			cwlog.DoLogCW("No softmod files found, stopping.")
			return
		}

		/* Add old save files into zip */
		path := cfg.GetSavesFolder()

		newZipFile, err := os.Create(path + constants.TempSaveName)
		if err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to create temp save.")
			return
		}
		defer newZipFile.Close()

		zipWriter := zip.NewWriter(newZipFile)
		defer zipWriter.Close()

		for _, file := range zipFiles {
			fh := new(zip.FileHeader)
			fh.Name = file.Name
			fh.UncompressedSize64 = uint64(len(file.Data))

			writer, err := zipWriter.CreateHeader(fh)
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to create blank file in zip.")
				continue
			}

			_, err = writer.Write(file.Data)
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to copy file data into zip.")
				continue
			}
		}

		err = os.Rename(path+constants.TempSaveName, fileName)
		if err != nil {
			cwlog.DoLogCW("Couldn't rename softmod temp save.")
			return
		}
		cwlog.DoLogCW("SoftMod injected.")

	}
}
