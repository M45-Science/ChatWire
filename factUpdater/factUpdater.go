package factUpdater

import (
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func getFactorioVersion(info *InfoData) error {

	if info.Version != "" {
		return nil
	}

	if err := factBinExist(); err != nil {
		return err
	}

	if fact.FactIsRunning {
		return fmt.Errorf("Factorio is already running.")
	}

	glob.FactorioLock.Lock()
	defer glob.FactorioLock.Unlock()

	var tempargs []string
	tempargs = append(tempargs, "--version")

	var cmd *exec.Cmd = exec.Command(fact.GetFactorioBinary(), tempargs...)

	//cwlog.DoLogCW("Executing: " + fact.GetFactorioBinary() + " " + strings.Join(tempargs, " "))

	// Run the command and capture its output
	output, err := cmd.Output()
	if err != nil {
		cwlog.DoLogCW("getFactorioVersion: Factorio failed to start: %s", err)
		return err
	}

	fversion, fbuild, fdistro, err := parseFactorioVersion(string(output))
	if err != nil {
		return err
	}

	saveFoundVersion(info, fversion, fbuild, fdistro)
	return nil
}

func factBinExist() error {

	//Check if factorio binary exists
	found, _, err := fileExistsSize(fact.GetFactorioBinary())
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("factorio binary was not found at: %v", fact.GetFactorioBinary())
	}

	return nil
}

func checkInstallTar(data []byte) error {

	var fileChecks = []checkInstallData{
		{name: "factorio/data/eula.txt"},
		{name: "factorio/data/licenses.txt"},
		{name: "factorio/data/credits.txt"},
		{name: "factorio/data/changelog.txt"},
		{name: "factorio/data/core/backers.json"},
		{name: "factorio/data/core/data.lua"},
		{name: "factorio/data/core/info.json"},
	}
	tr := tar.NewReader(bytes.NewReader(data))

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if header == nil {
			continue
		}

		if tar.TypeReg == header.Typeflag {
			for i, item := range fileChecks {
				if item.name == header.Name {
					fileChecks[i].good = true
					break
				}
			}
		}
	}

	for _, item := range fileChecks {
		if !item.good {
			return fmt.Errorf("file '%v' missing from install package", item.name)
		}
	}

	return nil
}

func parseFactorioVersion(input string) (versionInts, string, string, error) {
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		words := strings.Split(line, " ")
		numWords := len(words) - 1

		if numWords >= 5 {
			if words[0] == "Version:" {
				fversion, err := parseVersionString(words[1])
				fdistro := strings.TrimSuffix(words[4], ",")
				fbuild := strings.TrimSuffix(words[5], ")")

				return fversion, fbuild, fdistro, err
			} else if words[1] == "Goodbye" {
				return versionInts{}, "", "", fmt.Errorf("factorio quit")
			}
		}
	}

	cwlog.DoLogCW("Unexpected error: %v", input)
	return versionInts{}, "", "", fmt.Errorf("unexpected error")
}

func saveFoundVersion(info *InfoData, fversion versionInts, fbuild, fdistro string) {
	if info.Version == "" {
		info.VersInt = fversion
	}
	if info.Build == "" {
		info.Build = fbuild
	}
	if info.Distro == "" {
		info.Distro = fdistro
	}
}
