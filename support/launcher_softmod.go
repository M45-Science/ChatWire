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

func keepSaveEntry(name, folderName string) bool {
	if path.Dir(name) != folderName {
		return false
	}

	fileName := path.Base(name)
	return strings.HasPrefix(fileName, "level.dat") ||
		strings.HasSuffix(fileName, ".json") ||
		strings.HasSuffix(fileName, ".dat") ||
		strings.EqualFold(fileName, "level-init.dat") ||
		strings.EqualFold(fileName, "level.datmetadata")
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
		for _, f := range archive.File {
			if keepSaveEntry(f.Name, folderName) {
				file, err := f.Open()
				if err != nil {
					cwlog.DoLogCW("sm-inject: unable to open " + f.Name)
				} else {
					data, rerr := io.ReadAll(file)
					cerr := file.Close()

					dlen := uint64(len(data))
					if rerr != nil && rerr != io.EOF {
						cwlog.DoLogCW("Unable to read file: " + f.Name)
						continue
					} else if cerr != nil {
						cwlog.DoLogCW("Unable to close file: " + f.Name)
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
		if err := archive.Close(); err != nil {
			cwlog.DoLogCW("sm-inject: unable to close save game: %v", err)
			return
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

		tempSaveName := path + constants.TempSaveName
		newZipFile, err := os.Create(tempSaveName)
		if err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to create temp save.")
			return
		}
		defer os.Remove(tempSaveName)

		zipWriter := zip.NewWriter(newZipFile)

		for _, file := range zipFiles {
			fh := new(zip.FileHeader)
			fh.Name = file.Name
			fh.UncompressedSize64 = uint64(len(file.Data))

			writer, err := zipWriter.CreateHeader(fh)
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to create blank file in zip.")
				_ = zipWriter.Close()
				_ = newZipFile.Close()
				return
			}

			_, err = writer.Write(file.Data)
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to copy file data into zip.")
				_ = zipWriter.Close()
				_ = newZipFile.Close()
				return
			}
		}

		if err := zipWriter.Close(); err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to finalize temp save: %v", err)
			_ = newZipFile.Close()
			return
		}
		if err := newZipFile.Sync(); err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to sync temp save: %v", err)
			_ = newZipFile.Close()
			return
		}
		if err := newZipFile.Close(); err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to close temp save: %v", err)
			return
		}

		err = os.Rename(tempSaveName, fileName)
		if err != nil {
			cwlog.DoLogCW("Couldn't rename softmod temp save.")
			return
		}
		cwlog.DoLogCW("SoftMod injected.")

	}
}
