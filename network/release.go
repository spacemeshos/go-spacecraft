package network

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v41/github"
	"github.com/spacemeshos/go-spacecraft/gcp"
	"golang.org/x/oauth2"
)

func ReleaseNetwork() error {
	tempDir := os.TempDir() + "go-spacecraft/"
	err := os.RemoveAll(tempDir)
	if err != nil {
		return err
	}

	err = os.Mkdir(tempDir, 0o755)

	if err != nil {
		return err
	}

	goSpacemeshBuildsBucket := "https://storage.googleapis.com/go-spacemesh-release-builds/"

	osList := []string{"Windows", "macOS", "Linux"}

	configFile, err := gcp.ReadConfig(config.NetworkName)
	if err != nil {
		return err
	}

	for _, osBuild := range osList {
		fmt.Println("started downloading: " + osBuild + ".zip")
		downloadFile(tempDir+osBuild+".zip", goSpacemeshBuildsBucket+config.GoSmReleaseVersion+"/"+osBuild+".zip")
		fmt.Println("finished downloading: " + osBuild + ".zip")

		fmt.Println("unzipping: " + osBuild + ".zip")
		unzip(tempDir+osBuild+".zip", tempDir)

		err := os.RemoveAll(tempDir + osBuild + ".zip")
		if err != nil {
			return err
		}

		fmt.Println("writting config file to " + osBuild)

		f, err := os.Create(tempDir + osBuild + "/config.json")
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.WriteString(configFile)

		if err != nil {
			return err
		}

		fmt.Println("zipping: " + osBuild + ".zip")
		zipDir(tempDir+osBuild, tempDir+osBuild+".zip")

		fmt.Println("uploading started for file: " + osBuild + ".zip")
		gcp.UploadReleaseBuild(osBuild+".zip", tempDir+osBuild+".zip")
		fmt.Println("uploading finished for file: " + osBuild + ".zip")
		fmt.Println("download url: " + "https://storage.googleapis.com/spacemesh-release-builds/" + config.GoSmReleaseVersion + "/" + osBuild + ".zip")
	}

	err = os.RemoveAll(tempDir)

	if err != nil {
		return err
	}

	draft := true
	preRelease := true
	name := strings.ToUpper(config.NetworkName)
	tagName := config.GoSmReleaseVersion
	body := fmt.Sprintf(`
## Apps

Windows: https://storage.googleapis.com/spacemesh-release-builds/%s/Windows.zip
macOS: https://storage.googleapis.com/spacemesh-release-builds/%s/macOS.zip
Linux: https://storage.googleapis.com/spacemesh-release-builds/%s/Linux.zip

## Config File

https://storage.googleapis.com/spacecraft-data/%s-archive/config.json
	`, config.GoSmReleaseVersion, config.GoSmReleaseVersion, config.GoSmReleaseVersion, config.NetworkName)

	input := &github.RepositoryRelease{
		Name:       &name,
		TagName:    &tagName,
		Body:       &body,
		Draft:      &draft,
		Prerelease: &preRelease,
	}

	tc := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GithubToken},
	))

	client := github.NewClient(tc)

	_, _, err = client.Repositories.CreateRelease(context.Background(), "spacemeshos", "sm-net", input)

	if err != nil {
		return err
	}

	return nil
}

func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func zipDir(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}
