package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	chelper "github.com/ArthurHlt/go-concourse-helper"
	"github.com/blang/semver"
	"github.com/jfrog/jfrog-cli-core/v2/artifactory/commands/generic"
	artutils "github.com/jfrog/jfrog-cli-core/v2/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/v2/common/spec"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/orange-cloudfoundry/artifactory-resource/model"
	"github.com/orange-cloudfoundry/artifactory-resource/utils"
)

type SemverFile struct {
	Path    string
	Version semver.Version
}

const (
	SEMVER_REGEX = `(v|-|_)?v?((?:\d+)\.?(?:\d+)?\.?(?:\d+)?(?:(?:-|\+)(?:dev|alpha|beta)(\.[0-9]+)?)?)`
)

type Check struct {
	cmd        *chelper.CheckCommand
	source     model.Source
	artdetails *config.ServerDetails
	spec       *spec.SpecFiles
}

func main() {
	check := &Check{
		cmd: chelper.NewCheckCommand(),
	}
	check.Run()
}
func (c *Check) Run() {
	cmd := c.cmd
	msg := c.cmd.Messager()
	c.source.Recursive = true
	err := cmd.Source(&c.source)

	msg.FatalIf("Error when parsing source from concourse", err)
	utils.OverrideLoggerArtifactory(c.source.LogLevel)
	err = utils.CheckReqParamsWithPattern(c.source)
	if err != nil {
		msg.Fatal(err.Error())
	}
	c.artdetails, err = utils.RetrieveArtDetails(c.source)
	if err != nil {
		msg.Fatal(err.Error())
	}
	builder := spec.NewBuilder()
	c.spec = builder.
		Pattern(c.source.Pattern).
		Target("").
		Props(c.source.Props).
		Regexp(c.source.Regexp).
		Recursive(c.source.Recursive).
		Flat(c.source.Flat).
		BuildSpec()

	origStdout := os.Stdout
	os.Stdout = os.Stderr
	results, err := c.Search()
	os.Stdout = origStdout
	msg.FatalIf("Error when trying to find latest file", err)
	versions, err := c.RetrieveVersions(results)
	msg.FatalIf("Error when retrieving versions", err)
	cmd.Send(versions)
}

func (c Check) Search() ([]artutils.SearchResult, error) {
	res := []artutils.SearchResult{}
	cmd := generic.NewSearchCommand()
	cmd.
		SetServerDetails(c.artdetails).
		SetSpec(c.spec)

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

func (c Check) RetrieveVersions(results []artutils.SearchResult) ([]chelper.Version, error) {
	versions := make([]chelper.Version, 0)
	if len(results) == 0 {
		return versions, nil
	}
	if c.source.Version == "" {
		for _, file := range results {
			versions = append(versions, chelper.Version{
				BuildNumber: file.Path,
			})
		}
		return versions, nil
	}
	semverPrevious := c.RetrieveSemverFilePrevious()
	if semverPrevious.Path != "" {
		versions = append(versions, chelper.Version{
			BuildNumber: semverPrevious.Path,
		})
	}
	rangeSem, err := c.RetrieveRange()
	if err != nil {
		return versions, err
	}
	semverFiles := c.ResultsToSemverFilesFiltered(results, rangeSem)
	versions = append(versions, c.SemverFilesToVersions(semverFiles)...)
	return versions, nil
}

func (c *Check) RetrieveRange() (semver.Range, error) {
	rangeSem, err := semver.ParseRange(c.SanitizeVersion(c.source.Version))
	if err != nil {
		return nil, errors.New("Error when trying to create semver range: " + err.Error())
	}
	semverPrevious := c.RetrieveSemverFilePrevious()
	if semverPrevious.Path != "" {
		prevRangeSem, _ := semver.ParseRange(">" + semverPrevious.Version.String())
		c.source.Version += " && >" + semverPrevious.Version.String()
		rangeSem = rangeSem.AND(prevRangeSem)
	}
	return rangeSem, nil
}

func (c Check) SemverFilesToVersions(semverFiles []SemverFile) []chelper.Version {
	sort.Slice(semverFiles, func(i, j int) bool {
		return semverFiles[i].Version.LT(semverFiles[j].Version)
	})
	versions := make([]chelper.Version, 0)
	for _, fileSemver := range semverFiles {
		versions = append(versions, chelper.Version{
			BuildNumber: fileSemver.Path,
		})
	}
	return versions
}

func (c Check) RetrieveSemverFilePrevious() SemverFile {
	semverFile, _ := c.SemverFromPath(c.cmd.Version().BuildNumber)
	return semverFile
}

func (c Check) ResultsToSemverFilesFiltered(results []artutils.SearchResult, rangeSem semver.Range) []SemverFile {
	msg := c.cmd.Messager()
	semverFiles := make([]SemverFile, 0)
	for _, file := range results {
		semverFile, err := c.SemverFromPath(file.Path)
		if err != nil {
			msg.Logln("[yellow]Error[reset] for file '[blue]%s[reset]': %s [reset]", file.Path, err.Error())
			continue
		}
		if !rangeSem(semverFile.Version) {
			msg.Logln(
				"[cyan]Skipping[reset] file '[blue]%s[reset]' with version '[blue]%s[reset]' because it doesn't satisfy range '[blue]%s[reset]' [reset]",
				file.Path,
				semverFile.Version.String(),
				c.source.Version,
			)
			continue
		}
		msg.Logln("[blue]Found[reset] valid file '[blue]%s[reset]' in version '[blue]%s[reset]' [reset]", file.Path, semverFile.Version.String())
		semverFiles = append(semverFiles, semverFile)
	}
	return semverFiles
}

func (c Check) SanitizeVersion(version string) string {
	splitVersion := strings.Split(version, ".")
	if len(splitVersion) == 1 {
		version += ".0.0"
	} else if len(splitVersion) == 2 {
		version += ".0"
	}
	return version
}

func (c Check) SemverFromPath(path string) (SemverFile, error) {
	if path == "" {
		return SemverFile{}, nil
	}
	pathSplitted := strings.Split(path, "/")
	file := pathSplitted[len(pathSplitted)-1]
	ext := filepath.Ext(file)
	if ext != "" {
		file = strings.TrimSuffix(file, ext)
	}
	r := regexp.MustCompile("(?i)" + SEMVER_REGEX)
	allMatch := r.FindAllStringSubmatch(file, -1)
	if len(allMatch) == 0 {
		return SemverFile{}, errors.New("Cannot find any semver in file.")
	}
	if len(allMatch[0]) < 3 {
		return SemverFile{}, errors.New("Cannot find any semver in file.")
	}
	versionFound := c.SanitizeVersion(allMatch[len(allMatch)-1][2])

	semverFound, err := semver.Make(versionFound)
	if err != nil {
		return SemverFile{}, err
	}
	return SemverFile{
		Path:    path,
		Version: semverFound,
	}, nil
}
