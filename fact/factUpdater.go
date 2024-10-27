package fact

import (
	"archive/zip"

	"ChatWire/cwlog"
)

func CheckIfNewer(ca, cb, cc int) bool {

	if FactorioVersionA > ca &&
		FactorioVersionB > cb &&
		FactorioVersionC > cc {
		return true
	}
	return false
}

/* Check if Factorio update zip is valid */
func CheckZip(filename string) bool {
	read, err := zip.OpenReader(filename)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return false
	}
	defer read.Close()

	for _, file := range read.File {
		data, err := file.Open()
		if err != nil {
			cwlog.DoLogCW(err.Error())
			return false
		}
		defer data.Close()
		if file.UncompressedSize64 > 1024 {
			return true
		}
	}

	return false
}
