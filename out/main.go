package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jfrog/jfrog-cli-artifactory/artifactory/commands/generic"
	artutils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	buildutils "github.com/jfrog/jfrog-cli-core/v2/common/build"
	"github.com/jfrog/jfrog-cli-core/v2/common/spec"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/orange-cloudfoundry/artifactory-resource/model"
	"github.com/orange-cloudfoundry/artifactory-resource/utils"
	"gopkg.in/yaml.v3"
)

type Out struct {
	source     model.Source
	params     model.OutParams
	artdetails *config.ServerDetails
}

func main() {
	request := model.OutRequest{}.Default()

	err := utils.RetrieveJsonRequest(&request)
	if err != nil {
		utils.Fatal("error when parsing object given by concourse: " + err.Error())
	}
	utils.OverrideLoggerArtifactory(request.Source.LogLevel)

	Out := &Out{
		source: request.Source,
		params: request.Params,
	}
	Out.Run()
}

func (c *Out) Run() {
	err := utils.CheckReqParams(c.source)
	if err != nil {
		utils.Fatal(err.Error())
	}

	c.artdetails, err = utils.RetrieveArtDetails(c.source)
	if err != nil {
		utils.Fatal(err.Error())
	}

	c.source.Repository = utils.AddTrailingSlashIfNeeded(c.source.Repository)
	props := c.mergeProps()
	toUpload := c.getUploadFiles()
	filesToSpec := c.filesToSpec(toUpload, props)

	// upload
	for _, s := range filesToSpec.Files {
		utils.Log("uploading '%s' to '%s'...", s.Pattern, c.source.Repository)
	}
	startDl := time.Now()
	origStdout := os.Stdout
	os.Stdout = os.Stderr
	meta, err := c.upload(filesToSpec)
	os.Stdout = origStdout
	if err != nil {
		utils.Fatal("error when uploading: %s", err)
	}
	elapsed := time.Since(startDl)
	utils.Log("finished uploading files to '%s'", c.source.Repository)

	// use last file as version info
	filter := utils.NewFilter(c.source.Filter)
	ts := time.Now().Format(utils.TS_FORMAT)
	version := model.Version{}
	for _, file := range toUpload {
		_, key := filter.Match(file, ts)
		version = model.Version{
			File:    filepath.Join(c.source.Repository, file),
			Version: key,
		}
	}

	meta = append(meta, model.Metadata{
		Name:  "elapsed",
		Value: elapsed.String(),
	})

	err = utils.SendJsonResponse(model.Response{
		Metadata: meta,
		Version:  version,
	})
	if err != nil {
		utils.Log(fmt.Sprintf("error sending request to artifactory: %s", err.Error()))
	}
}

func (c Out) getFilePath(p string) string {
	src := utils.AddTrailingSlashIfNeeded(utils.BaseDirectory())
	src += utils.RemoveStartingSlashIfNeeded(p)
	return src
}

func (c Out) upload(spec *spec.SpecFiles) ([]model.Metadata, error) {
	cmd := generic.NewUploadCommand()
	cmd.SetUploadConfiguration(&artutils.UploadConfiguration{
		Threads: c.source.Threads,
	}).SetBuildConfiguration(&buildutils.BuildConfiguration{})

	cmd.
		SetServerDetails(c.artdetails).
		SetDetailedSummary(true).
		SetSpec(spec)

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	return utils.TransfertDetailsToMeta(cmd.Result()), nil
}

func (c Out) getUploadFiles() []string {
	files, err := os.ReadDir(filepath.Join(utils.BaseDirectory(), c.params.Directory))
	if err != nil {
		utils.Fatal(fmt.Sprintf("could not list files in directory '%s': %s", c.params.Directory, err))
	}

	filter := utils.NewFilter(c.source.Filter)
	ts := time.Now().Format(utils.TS_FORMAT)
	res := []string{}
	for _, file := range files {
		match, _ := filter.Match(file.Name(), ts)
		if !match {
			continue
		}
		res = append(res, file.Name())
	}

	if len(res) == 0 {
		utils.Fatal(fmt.Sprintf("could find any file matching filter '%s' in directory '%s'", c.source.Filter, c.params.Directory))
	}

	return res
}

func (c Out) filesToSpec(files []string, props model.Properties) *spec.SpecFiles {
	res := &spec.SpecFiles{
		Files: []spec.File{},
	}

	for _, file := range files {
		absPath := filepath.Join(utils.BaseDirectory(), c.params.Directory, file)
		builder := spec.NewBuilder()
		buildSpec := builder.
			Pattern(absPath).
			Target(c.source.Repository).
			Props(props.String()).
			Flat(true).
			BuildSpec()
		res.Files = append(res.Files, buildSpec.Files...)
	}

	return res
}

func (c Out) mergeProps() model.Properties {
	props := model.Properties{}
	props.Merge(c.source.Props)
	props.Merge(c.params.Props)

	if c.params.PropsFilename != "" {
		fProps := model.Properties{}

		content, err := os.ReadFile(c.getFilePath(c.params.PropsFilename))
		if err != nil {
			utils.Fatal(fmt.Sprintf("could not read properties from file '%s': %s", c.params.PropsFilename, err))
		}
		err = yaml.Unmarshal(content, fProps)
		if err != nil {
			utils.Fatal(fmt.Sprintf("invalid yaml format in file '%s': %s", c.params.PropsFilename, err))
		}
		props.Merge(fProps)
	}
	return props
}
