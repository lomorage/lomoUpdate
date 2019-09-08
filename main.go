package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus"

	"path/filepath"

	"archive/zip"
	"crypto/sha256"
	"io/ioutil"

	"time"

	"os/exec"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// LomoUpdateVersion lomoUpdate version auto generated
const LomoUpdateVersion = "2019_09_08.15_55_34.0.838cfd7"

type platform struct {
	URL      string
	SHA256   string
	Version  string
	PreCmds  []string
	PostCmds []string
}

type releases map[string]platform

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s\n", c.App.Version)
	}

	app := cli.NewApp()

	app.Version = LomoUpdateVersion
	app.Usage = "personal photo backup solution backend daemon"
	app.Email = "leslie.qiwa@gmail.com"

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "app-dir, a",
			Usage: "app directory to uncompress lomorage zip",
			Value: dir,
		},
		cli.StringFlag{
			Name:  "backup-dir, b",
			Usage: "directory to back up downloaded zip file and old release",
		},
		cli.StringFlag{
			Name:  "curr-version, c",
			Usage: "current version of lomorage app",
			Value: dir,
		},
		cli.StringFlag{
			Name:  "url, u",
			Usage: "url for release json",
			Value: "http://lomorage.github.io/release.json",
		},

		cli.StringFlag{
			Name:  "precmd, prc",
			Usage: "PreCmd for upgrading",
			Value: "c:/stopLomoagent.bat",
		},

		cli.StringFlag{
			Name:  "precmdarg, prca",
			Usage: "PreCmd args for upgrading",
			Value: "",
		},

		cli.StringFlag{
			Name:  "postcmd, psc",
			Usage: "PostCmd for upgrading",
			Value: "c:/startLomoagent.bat",
		},

		cli.StringFlag{
			Name:  "postcmdarg, psca",
			Usage: "PostCmdArgs for upgrading",
			Value: "c:/lomoagent.exe",
		},
	}

	app.Action = bootService

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func bootService(ctx *cli.Context) error {
	if ctx.String("app-dir") == "" {
		return errors.New("invalid app dir")
	}
	if ctx.String("curr-version") == "" {
		return errors.New("invalid current version")
	}
	if ctx.String("url") == "" {
		return errors.New("invalid url")
	}
	p, err := downloadReleaseMeta(ctx.String("url"))
	if err != nil {
		fmt.Println("downloadReleaseMeta error")
		return err
	}
	if p.Version == ctx.String("curr-version") {
		fmt.Println("No new version, skip upgrade")
		return nil
	}

	fmt.Println("Got new version, start upgrade")

	tempRoot := ctx.String("backup-dir")
	if tempRoot == "" {
		tempRoot, err = ioutil.TempDir("", "lomod-temp")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempRoot)
	}

	f, err := downloadReleaseBin(p.URL, p.SHA256, tempRoot)
	if err != nil {
		return err
	}
	defer os.Remove(f)

	tempUncompress := filepath.Join(tempRoot, "uncompress")
	if err := uncompress(f, tempUncompress); err != nil {
		return err
	}

	tempPreCmd := ctx.String("precmd")
	tempPreCmdArg := ctx.String("precmdarg")

	fmt.Println("start preUpgrade...")

	if err := preUpgrade(tempPreCmd, tempPreCmdArg); err != nil {
		// return err
		fmt.Println("preUpgrade fail...")
	}

	fmt.Println("start upgrade...")
	if err := upgrade(ctx.String("app-dir"), tempRoot, tempUncompress); err != nil {
		fmt.Println("upgrade fail...")
		// return err
	}

	tempPostCmd := ctx.String("postcmd")
	tempPostCmdArg := ctx.String("postcmdarg")

	fmt.Println("start postUpgrade...")
	return postUpgrade(tempPostCmd, tempPostCmdArg)
}

func downloadReleaseMeta(url string) (*platform, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	v := make(releases)
	err = d.Decode(&v)
	if err != nil {
		return nil, err
	}

	p, ok := v[runtime.GOOS]
	if !ok {
		return nil, errors.Errorf("Unsupported platform: %s", runtime.GOOS)
	}
	return &p, nil
}

func downloadReleaseBin(url, expectedSHA, tmpdir string) (string, error) {
	tmpfile, err := ioutil.TempFile(tmpdir, "")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	sha := sha256.New()
	mw := io.MultiWriter(sha, tmpfile)
	size, err := io.Copy(mw, resp.Body)
	if err != nil {
		return "", err
	}
	if size == 0 {
		return "", errors.New("Empty file size")
	}

	// sha256Temp := fmt.Sprintf("%x", sha.Sum(nil))
	// fmt.Println("the file's SHA256=", sha256Temp)

	if strings.ToUpper(fmt.Sprintf("%x", sha.Sum(nil))) != strings.ToUpper(expectedSHA) {
		return "", errors.New("Hash not match")
	}

	return tmpfile.Name(), nil
}

func uncompress(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		fpath := filepath.Join(dst, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			// Make Folder
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), f.Mode()); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return nil
		}
		if err := rc.Close(); err != nil {
			return err
		}
		if err := outFile.Close(); err != nil {
			return err
		}
	}
	return nil
}

func preUpgrade(preCmd string, preCmdArg string) error {
	cmd := exec.Command(preCmd, preCmdArg)

	//log.Printf("Running command and waiting for it to finish...")
	err := cmd.Run()
	return err
}

func upgrade(appDir, bakDir, downloadDir string) error {
	now := time.Now()
	if err := os.Rename(appDir,
		filepath.Join(bakDir, fmt.Sprintf("lomod-bak-%d%02d%02d_%02d%02d%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second()))); err != nil {
		return err
	}

	// move new one to specified app-dir
	return os.Rename(downloadDir, appDir)
}

func postUpgrade(postCmd string, postCmdArg string) error {
	cmd := exec.Command(postCmd, postCmdArg)

	//log.Printf("Running command and waiting for it to finish...")
	err := cmd.Start()
	return err
}
