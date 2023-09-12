package main

import (
	"os"
	"sort"

	"github.com/jfrog/jfrog-cli-core/v2/artifactory/commands/generic"
	artutils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/common/spec"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/orange-cloudfoundry/artifactory-resource/model"
	"github.com/orange-cloudfoundry/artifactory-resource/utils"
)

type Check struct {
	source     model.Source
	version    model.Version
	artdetails *config.ServerDetails
}

type Match struct {
	artutils.SearchResult
	key string
}

func main() {
	request := model.CheckRequest{}.Default()

	err := utils.RetrieveJsonRequest(&request)
	if err != nil {
		utils.Fatal("error when parsing object given by concourse: " + err.Error())
	}
	utils.OverrideLoggerArtifactory(request.Source.LogLevel)
	cmd := Check{
		source:  request.Source,
		version: request.Version,
	}
	cmd.Run()
}

func (c *Check) Run() {
	err := utils.CheckReqParamsWithPattern(c.source)
	if err != nil {
		utils.Fatal(err.Error())
	}
	c.artdetails, err = utils.RetrieveArtDetails(c.source)
	if err != nil {
		utils.Fatal(err.Error())
	}

	c.source.Repository = utils.AddTrailingSlashIfNeeded(c.source.Repository)

	builder := spec.NewBuilder()
	specFiles := builder.
		Pattern(c.source.Repository).
		Props(c.source.Props.String()).
		BuildSpec()

	origStdout := os.Stdout
	os.Stdout = os.Stderr
	results, err := c.search(specFiles)
	os.Stdout = origStdout
	if err != nil {
		utils.Fatal("error when trying to find latest file: %s", err)
	}

	matches := c.filter(results)

	versions := []model.Version{}
	for _, m := range matches {
		versions = append(versions, model.Version{
			Version: m.key,
			File:    m.Path,
		})
	}
	utils.SendJsonResponse(versions)
}

func (c Check) search(spec *spec.SpecFiles) ([]artutils.SearchResult, error) {
	res := []artutils.SearchResult{}
	cmd := generic.NewSearchCommand()
	cmd.
		SetServerDetails(c.artdetails).
		SetSpec(spec)

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	reader := cmd.Result().Reader()
	defer reader.Close()
	_, err = reader.Length()
	if err != nil {
		return nil, err
	}

	for val := new(artutils.SearchResult); reader.NextRecord(val) == nil; val = new(artutils.SearchResult) {
		res = append(res, *val)
	}

	return res, nil
}

func (c Check) filter(results []artutils.SearchResult) []Match {
	filter := utils.NewFilter(c.source.Filter)

	res := []Match{}
	for _, file := range results {
		match, key := filter.Match(file.Path, file.Modified)
		if !match {
			continue
		}
		if c.version.Version != "" && filter.Less(key, c.version.Version) {
			continue
		}
		res = append(res, Match{
			SearchResult: file,
			key:          key,
		})
	}

	// sort results, newest to oldest
	sort.Slice(res, func(i, j int) bool {
		return filter.Less(res[i].key, res[j].key)
	})

	return res
}
