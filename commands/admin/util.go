package admin

import (
	"archive/tar"
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/ulikunitz/xz"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
)

func installFactorio(s *discordgo.Session, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(s, i, "Status:", "Downloading Factorio server...")
	resp, err := http.Get(constants.FactHeadlessURL)

	if err != nil {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: "Unable to download Factorio server."})
		f := discordgo.WebhookParams{Embeds: elist, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)

		cwlog.DoLogCW(err.Error())
		return
	}
	defer resp.Body.Close()

	gzdata, err := io.ReadAll(resp.Body)
	if err != nil {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: "Unable to read the http response."})
		f := discordgo.WebhookParams{Embeds: elist, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)

		cwlog.DoLogCW(err.Error())
		return
	}

	var elist []*discordgo.MessageEmbed
	elist = append(elist, &discordgo.MessageEmbed{Title: "Status:", Description: "Downloaded, decompressing..."})
	f := discordgo.WebhookParams{Embeds: elist, Flags: 1 << 6}
	disc.FollowupResponse(s, i, &f)

	data, err := unXZData(gzdata)
	if err != nil {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: "The gzip data appears to be invalid."})
		f := discordgo.WebhookParams{Embeds: elist, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)

		cwlog.DoLogCW(err.Error())
		return
	}

	dest := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix + cfg.Local.Callsign + "/"

	err = untar(dest, data)
	if err != nil {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: "Unable to open tar archive."})
		f := discordgo.WebhookParams{Embeds: elist, Flags: 1 << 6}
		disc.FollowupResponse(s, i, &f)

		cwlog.DoLogCW(err.Error())
		return
	}

	os.Mkdir("factorio/saves", 0755)

	if err == nil {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Success:", Description: "Factorio server installed!"})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(s, i, &f)
		return
	}

}

func unXZData(data []byte) ([]byte, error) {
	r, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func untar(dst string, data []byte) error {

	tr := tar.NewReader(bytes.NewReader(data))

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0770)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
