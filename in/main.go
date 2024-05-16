package main

import (
	"fmt"
	buildutils "github.com/jfrog/jfrog-cli-core/v2/common/build"
	"os"
	"path/filepath"
	"time"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/commands/generic"
	artutils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/common/spec"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/orange-cloudfoundry/artifactory-resource/model"
	"github.com/orange-cloudfoundry/artifactory-resource/utils"
	"gopkg.in/yaml.v3"
)

type In struct {
	source     model.Source
	params     model.InParams
	version    model.Version
	artdetails *config.ServerDetails
	spec       *spec.SpecFiles
}

func main() {
	request := model.InRequest{}.Default()

	err := utils.RetrieveJsonRequest(&request)
	if err != nil {
		utils.Fatal("error when parsing object given by concourse: " + err.Error())
	}
	utils.OverrideLoggerArtifactory(request.Source.LogLevel)
	cmd := In{
		source:  request.Source,
		params:  request.Params,
		version: request.Version,
	}
	cmd.Run()
}

func (c *In) Run() {
	err := utils.CheckReqParams(c.source)
	if err != nil {
		utils.Fatal(err.Error())
	}

	c.artdetails, err = utils.RetrieveArtDetails(c.source)
	if err != nil {
		utils.Fatal(err.Error())
	}

	c.source.Repository = utils.AddTrailingSlashIfNeeded(c.source.Repository)
	dest := utils.AddTrailingSlashIfNeeded(filepath.Join(utils.BaseDirectory(), c.params.Destination))
	builder := spec.NewBuilder()
	c.spec = builder.
		Pattern(c.version.File).
		Target(dest).
		Flat(true).
		Props(c.source.Props.String()).
		BuildSpec()

	utils.Log("downloading '%s' to '%s'...", c.version.File, dest)
	startDl := time.Now()

	origStdout := os.Stdout
	os.Stdout = os.Stderr
	meta, err := c.download()
	os.Stdout = origStdout
	if err != nil {
		utils.Fatal("error when downloading: %s", err)
	}

	elapsed := time.Since(startDl)
	utils.Log("finished downloading '%s' to '%s'", c.version.File, dest)
	meta = append(meta, model.Metadata{
		Name:  "elapsed",
		Value: elapsed.String(),
	})

	if c.params.PropsFilename != "" {
		utils.Log("downloading properties for '%s' to '%s'...", c.version.File, c.params.PropsFilename)
		val := c.downloadProps(c.version.File, c.params.PropsFilename)
		utils.Log("finished downloading properties for '%s' to '%s'", c.version.File, c.params.PropsFilename)
		utils.Log("%s", val)
	}

	err = utils.SendJsonResponse(model.Response{
		Metadata: meta,
		Version:  c.version,
	})
	if err != nil {
		utils.Log(err.Error())
	}
}

func (c In) download() ([]model.Metadata, error) {
	cmd := generic.NewDownloadCommand()
	cmd.SetConfiguration(&artutils.DownloadConfiguration{
		Threads:      c.source.Threads,
		SplitCount:   c.params.SplitCount,
		MinSplitSize: int64(c.params.MinSplit),
	}).SetBuildConfiguration(&buildutils.BuildConfiguration{})

	cmd.
		SetServerDetails(c.artdetails).
		SetDetailedSummary(true).
		SetSpec(c.spec)

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	return utils.TransfertDetailsToMeta(cmd.Result()), nil
}

func (c In) downloadProps(remoteFile string, propsFilename string) string {
	builder := spec.NewBuilder()
	spc := builder.
		Pattern(remoteFile).
		Props(model.Properties{}.String()).
		BuildSpec()

	cmd := generic.NewSearchCommand()
	cmd.
		SetServerDetails(c.artdetails).
		SetSpec(spc)

	err := cmd.Run()
	if err != nil {
		utils.Fatal(fmt.Sprintf("unable to fetch properties for file '%s': %s", c.version.File, err))
	}

	reader := cmd.Result().Reader()
	defer reader.Close()
	_, err = reader.Length()
	if err != nil {
		utils.Fatal(fmt.Sprintf("error while reading properties for file '%s': %s", c.version.File, err))
	}

	if length, _ := reader.Length(); length != 1 {
		utils.Fatal(fmt.Sprintf("error: found more than one property set for '%s'", c.version.File))
	}

	for res := new(artutils.SearchResult); reader.NextRecord(res) == nil; {
		content, _ := yaml.Marshal(res.Props)
		path := filepath.Join(utils.BaseDirectory(), propsFilename)
		err = os.WriteFile(path, content, 0644)
		if err != nil {
			utils.Fatal(fmt.Sprintf("unable to write prop file '%s': %s", path, err))
		}
		// nolint:staticcheck
		return string(content)
	}
	return ""
}
