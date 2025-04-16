package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver"

	cmdutils "github.com/jfrog/jfrog-cli-core/v2/artifactory/commands/utils"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	artlog "github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/orange-cloudfoundry/artifactory-resource/model"
)

const (
	ART_SECURITY_FOLDER = "security/"
	TS_FORMAT           = "2006-01-02T15:04:05.000Z"
)

func CheckReqParamsWithPattern(source model.Source) error {
	if source.Repository == "" {
		return errors.New("you must provide a repository (e.g.: 'bucket/folder/')")
	}
	return CheckReqParams(source)
}

func CheckReqParams(source model.Source) error {
	if source.Url == "" {
		return errors.New("you must pass an url to artifactory")
	}
	if source.User == "" && source.ApiKey == "" {
		return errors.New("you must pass user/password pair or apiKey to authnticate over artifactory")
	}
	if _, err := regexp.Compile(source.Filter); err != nil {
		return fmt.Errorf("invalid filter '%s', must be valid regexp: %s", source.Filter, err)
	}
	return nil
}

func RetrieveArtDetails(source model.Source) (*config.ServerDetails, error) {
	err := createCert(source.CACert)
	if err != nil {
		return nil, err
	}
	sshKeyPath, err := createSshKeyPath(source.SshKey)
	return &config.ServerDetails{
		ArtifactoryUrl: AddTrailingSlashIfNeeded(source.Url),
		Url:            AddTrailingSlashIfNeeded(source.Url),
		User:           source.User,
		Password:       source.Password,
		SshKeyPath:     sshKeyPath,
	}, err
}

func AddTrailingSlashIfNeeded(path string) string {
	if path != "" && !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func RemoveStartingSlashIfNeeded(path string) string {
	if path != "" && strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	return path
}

func createCert(caCert string) error {
	if caCert == "" {
		return nil
	}
	confPath, err := coreutils.GetJfrogHomeDir()
	if err != nil {
		return err
	}
	securityPath := confPath + ART_SECURITY_FOLDER
	if err := os.MkdirAll(securityPath, os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(securityPath+"cert.pem", []byte(caCert), 0644)
}

func createSshKeyPath(sshKey string) (string, error) {
	if sshKey == "" {
		return "", nil
	}
	file, err := os.CreateTemp(os.TempDir(), "ssh-key")
	if err != nil {
		return "", err
	}
	_, err = file.WriteString(sshKey)
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Dir(file.Name()))
}

func OverrideLoggerArtifactory(logLevel string) {
	lvl := artlog.INFO
	if strings.ToUpper(logLevel) == "ERROR" {
		lvl = artlog.ERROR
	} else if strings.ToUpper(logLevel) == "DEBUG" {
		lvl = artlog.DEBUG
	}
	logger := artlog.NewLogger(lvl, os.Stderr)
	artlog.SetLogger(logger)
}

func HashFile(path string, hasher hash.Hash) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer CloseAndLogError(file)
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func TransfertDetailsToMeta(result *cmdutils.Result) []model.Metadata {
	metadata := []model.Metadata{}
	if result != nil && result.Reader() != nil {
		reader := result.Reader()
		defer CloseAndLogError(reader)
		for d := new(clientutils.FileTransferDetails); reader.NextRecord(d) == nil; d = new(clientutils.FileTransferDetails) {
			if val, err := HashFile(d.SourcePath, sha1.New()); err == nil {
				metadata = append(metadata, model.Metadata{
					Name:  "sha1",
					Value: val,
				})
			}
			if val, err := HashFile(d.SourcePath, md5.New()); err == nil {
				metadata = append(metadata, model.Metadata{
					Name:  "md5",
					Value: val,
				})
			}
			if d.Sha256 != "" {
				metadata = append(metadata, model.Metadata{
					Name:  "sha256",
					Value: d.Sha256,
				})
			}
		}
	}
	return metadata
}

type Filter struct {
	re    *regexp.Regexp
	index int
	mode  string
}

func NewFilter(filter string) *Filter {
	f := &Filter{
		re:    regexp.MustCompile(filter),
		index: -1,
		mode:  "ts",
	}
	for _, key := range []string{"version", "asc", "desc"} {
		if idx := f.re.SubexpIndex(key); idx != -1 {
			f.mode = key
			f.index = idx
			break
		}
	}
	return f
}

func (f *Filter) Match(name string, modified_at string) (bool, string) {
	matches := f.re.FindStringSubmatch(name)
	if matches == nil {
		return false, ""
	}
	key := ""
	if f.mode != "ts" {
		key = matches[f.index]
	} else {
		key = modified_at
	}
	return key != "", key
}

func (f *Filter) Less(v1 string, v2 string) bool {
	switch f.mode {
	case "ts":
		ts1, err := time.Parse(TS_FORMAT, v1)
		if err != nil {
			return false
		}
		ts2, err := time.Parse(TS_FORMAT, v2)
		if err != nil {
			return true
		}
		return ts1.Before(ts2)

	case "asc":
		return strings.Compare(v1, v2) == -1

	case "desc":
		return strings.Compare(v2, v1) == -1

	case "version":
		sv1, err := semver.NewVersion(v1)
		if err != nil {
			return false
		}
		sv2, err := semver.NewVersion(v2)
		if err != nil {
			return true
		}
		return sv1.Compare(sv2) == -1
	}
	return false
}

func BaseDirectory() string {
	directory, _ := os.Getwd()
	if len(os.Args) >= 2 {
		directory = os.Args[1]
	}
	return directory
}

func Log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func Fatal(message string, args ...interface{}) {
	Log(message, args...)
	os.Exit(1)
}

func RetrieveJsonRequest(v interface{}) error {
	return json.NewDecoder(os.Stdin).Decode(v)
}

func SendJsonResponse(v interface{}) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}

// Utility function to close an io.Closer and log errors without returning them
func CloseAndLogError(closer io.Closer) {
	if closer == nil {
		return
	}

	// Attempt to close the resource (e.g., an HTTP response or a file).
	// If an error occurs during the close operation, the error is captured.
	if err := closer.Close(); err != nil {
		fmt.Printf("Error closing resource: %v", err)
	}
}
