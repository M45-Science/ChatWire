package support

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"ChatWire/cfg"
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
func readFolder(path string, sdir string) ([]zipFilesData, error) {

	var zipFiles []zipFilesData

	/* Get all softmod files */
	sFiles, err := os.ReadDir(path)
	if err != nil {
		cwlog.DoLogCW("Unable to read softmod folder.")
		return nil, err
	}

	for _, file := range sFiles {
		if !file.IsDir() {
			dat, err := os.ReadFile(path + "/" + file.Name())
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to read softmod files.")
				return nil, err
			}

			zipFiles = append(zipFiles, zipFilesData{Name: sdir + "/" + file.Name(), Data: dat})
		} else {
			tfiles, err := readFolder(path+"/"+file.Name(), sdir+"/"+file.Name())
			if err != nil {
				return nil, err
			}
			zipFiles = append(zipFiles, tfiles...)
		}
	}

	return zipFiles, nil
}

/* Insert our softmod files into the save zip */
func injectSoftMod(fileName, folderName string) error {
	var zipFiles []zipFilesData

	/* Read needed files from existing save */
	archive, errz := zip.OpenReader(fileName)
	if errz != nil {
		cwlog.DoLogCW("sm-inject: unable to open save game.")
		return fmt.Errorf("SoftMod injection failed for %s", fileName)
	} else {
		for _, f := range archive.File {
			if keepSaveEntry(f.Name, folderName) {
				file, err := f.Open()
				if err != nil {
					cwlog.DoLogCW("sm-inject: unable to open " + f.Name)
					_ = archive.Close()
					return fmt.Errorf("open save entry %s: %w", f.Name, err)
				} else {
					data, rerr := io.ReadAll(file)
					cerr := file.Close()

					dlen := uint64(len(data))
					if rerr != nil && rerr != io.EOF {
						cwlog.DoLogCW("Unable to read file: " + f.Name)
						_ = archive.Close()
						return fmt.Errorf("read save entry %s", f.Name)
					} else if cerr != nil {
						cwlog.DoLogCW("Unable to close file: " + f.Name)
						_ = archive.Close()
						return fmt.Errorf("read save entry %s", f.Name)
					} else if dlen != f.UncompressedSize64 {
						sbuf := fmt.Sprintf("%v vs %v", dlen, f.UncompressedSize64)
						cwlog.DoLogCW("Sizes did not match: " + f.Name + ", " + sbuf)
						_ = archive.Close()
						return fmt.Errorf("size mismatch for save entry %s", f.Name)
					} else {
						tmp := zipFilesData{Name: f.Name, Data: data}
						zipFiles = append(zipFiles, tmp)
					}
				}
			}
		}
		if err := archive.Close(); err != nil {
			cwlog.DoLogCW("sm-inject: unable to close save game: %v", err)
			return fmt.Errorf("SoftMod injection failed for %s", fileName)
		}

		/* Read files in from softmod */
		blackList := []string{"img-source", "out"}                   /* Wildcard exclude */
		allowList := []string{"README.md", "preview.jpg", "LICENSE"} /* Always include */
		allowExt := []string{".lua", ".png", ".cfg"}

		tfiles, err := readFolder(cfg.Local.Options.SoftModOptions.SoftModPath, folderName)
		if err != nil {
			return fmt.Errorf("read SoftMod payload: %w", err)
		}
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
		hasControl := false
		for _, f := range addFiles {
			if f.Name == folderName+"/control.lua" && len(f.Data) > 0 {
				hasControl = true
			}
		}
		if !hasControl {
			return fmt.Errorf("SoftMod payload requires a nonempty control.lua")
		}
		zipFiles = append(zipFiles, addFiles...)

		numFiles := len(zipFiles)
		if numFiles <= 0 {
			cwlog.DoLogCW("No softmod files found, stopping.")
			return fmt.Errorf("SoftMod injection failed for %s", fileName)
		}

		/* Add old save files into zip */
		newZipFile, err := os.CreateTemp(filepath.Dir(fileName), ".softmod-*.zip")
		if err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to create temp save.")
			return fmt.Errorf("SoftMod injection failed for %s", fileName)
		}
		tempSaveName := newZipFile.Name()
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
				return fmt.Errorf("SoftMod injection failed for %s", fileName)
			}

			_, err = writer.Write(file.Data)
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to copy file data into zip.")
				_ = zipWriter.Close()
				_ = newZipFile.Close()
				return fmt.Errorf("SoftMod injection failed for %s", fileName)
			}
		}

		if err := zipWriter.Close(); err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to finalize temp save: %v", err)
			_ = newZipFile.Close()
			return fmt.Errorf("SoftMod injection failed for %s", fileName)
		}
		if err := newZipFile.Sync(); err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to sync temp save: %v", err)
			_ = newZipFile.Close()
			return fmt.Errorf("SoftMod injection failed for %s", fileName)
		}
		if err := newZipFile.Close(); err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to close temp save: %v", err)
			return fmt.Errorf("SoftMod injection failed for %s", fileName)
		}

		err = os.Rename(tempSaveName, fileName)
		if err != nil {
			cwlog.DoLogCW("Couldn't rename softmod temp save.")
			return fmt.Errorf("SoftMod injection failed for %s", fileName)
		}
		cwlog.DoLogCW("SoftMod injected.")

	}
	return nil
}
