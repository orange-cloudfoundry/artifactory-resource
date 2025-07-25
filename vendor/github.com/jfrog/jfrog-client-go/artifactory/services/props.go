package services

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/jfrog/gofrog/parallel"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/auth"
	"github.com/jfrog/jfrog-client-go/http/jfroghttpclient"
	clientutils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/content"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type PropsService struct {
	client     *jfroghttpclient.JfrogHttpClient
	ArtDetails auth.ServiceDetails
	Threads    int
}

func NewPropsService(client *jfroghttpclient.JfrogHttpClient) *PropsService {
	return &PropsService{client: client}
}

func (ps *PropsService) GetArtifactoryDetails() auth.ServiceDetails {
	return ps.ArtDetails
}

func (ps *PropsService) SetArtifactoryDetails(rt auth.ServiceDetails) {
	ps.ArtDetails = rt
}

func (ps *PropsService) IsDryRun() bool {
	return false
}

func (ps *PropsService) GetThreads() int {
	return ps.Threads
}

func (ps *PropsService) SetProps(propsParams PropsParams) (int, error) {
	log.Info("Setting properties...")
	var err error
	var totalSuccess int
	if propsParams.RepoOnly {
		totalSuccess, err = ps.performRequestForRepoOnly(propsParams, false)
	} else {
		totalSuccess, err = ps.performRequest(propsParams, false)
	}
	if err == nil {
		log.Info("Done setting properties.")
	}
	return totalSuccess, err
}

func (ps *PropsService) DeleteProps(propsParams PropsParams) (int, error) {
	log.Info("Deleting properties...")
	var err error
	var totalSuccess int
	if propsParams.RepoOnly {
		totalSuccess, err = ps.performRequestForRepoOnly(propsParams, true)
	} else {
		totalSuccess, err = ps.performRequest(propsParams, true)
	}
	if err == nil {
		log.Info("Done deleting properties.")
	}
	return totalSuccess, err
}

type PropsParams struct {
	Reader   *content.ContentReader
	Props    string
	RepoOnly bool
}

func (sp *PropsParams) GetReader() *content.ContentReader {
	return sp.Reader
}

func (sp *PropsParams) GetProps() string {
	return sp.Props
}

func (ps *PropsService) getEncodedParam(propsParams PropsParams, isDelete bool) (string, error) {
	var encodedParam string
	if !isDelete {
		props, err := utils.ParseProperties(propsParams.GetProps())
		if err != nil {
			return "", err
		}
		encodedParam = props.ToEncodedString(true)
	} else {
		propList := strings.Split(propsParams.GetProps(), ",")
		for _, prop := range propList {
			encodedParam += url.QueryEscape(prop) + ","
		}
		// Remove trailing comma
		encodedParam = strings.TrimSuffix(encodedParam, ",")
	}
	return encodedParam, nil
}

func (ps *PropsService) actionTypeBasedOnIsDeleteFlag(isDelete bool, action *func(string, string, string) (*http.Response, []byte, error)) {
	if isDelete {
		*action = ps.sendDeleteRequest
	} else {
		*action = ps.sendPutRequest
	}
}

func (ps *PropsService) addOrDeletePropertiesForRepo(propsParams PropsParams, isDelete bool, encodedParam string) (int, error) {
	// Determine which action to perform (PUT or DELETE).
	var action func(string, string, string) (*http.Response, []byte, error)
	ps.actionTypeBasedOnIsDeleteFlag(isDelete, &action)

	reader := propsParams.GetReader()
	defer reader.Reset()
	resultItem := new(utils.ResultItem)
	if reader.NextRecord(resultItem) == nil {
		repoName := resultItem.Repo
		logMsgPrefix := clientutils.GetLogMsgPrefix(0, ps.IsDryRun())

		storageAPI := path.Join("api", "storage", repoName)
		setPropertiesURL, err := clientutils.BuildUrl(ps.GetArtifactoryDetails().GetUrl(), storageAPI, make(map[string]string))
		if err != nil {
			return 0, err
		}
		setPropertiesURL += "?properties=" + encodedParam + "&recursive=0"

		resp, body, err := action(logMsgPrefix, repoName, setPropertiesURL)
		if err != nil {
			return 0, err
		}
		if err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusNoContent); err != nil {
			return 0, err
		}

		return 1, nil
	}

	if err := reader.GetError(); err != nil {
		return 0, err
	}

	return 0, nil
}

func (ps *PropsService) performRequestForRepoOnly(propsParams PropsParams, isDelete bool) (int, error) {
	encodedParam, err := ps.getEncodedParam(propsParams, isDelete)
	if err != nil {
		return 0, err
	}

	return ps.addOrDeletePropertiesForRepo(propsParams, isDelete, encodedParam)
}

func (ps *PropsService) performRequest(propsParams PropsParams, isDelete bool) (int, error) {
	encodedParam, err := ps.getEncodedParam(propsParams, isDelete)
	if err != nil {
		return 0, err
	}
	var action func(string, string, string) (*http.Response, []byte, error)
	ps.actionTypeBasedOnIsDeleteFlag(isDelete, &action)
	successCounters := make([]int, ps.GetThreads())
	producerConsumer := parallel.NewBounedRunner(ps.GetThreads(), false)
	errorsQueue := clientutils.NewErrorsQueue(1)
	reader := propsParams.GetReader()
	go func() {
		for resultItem := new(utils.ResultItem); reader.NextRecord(resultItem) == nil; resultItem = new(utils.ResultItem) {
			relativePath := resultItem.GetItemRelativePath()
			setPropsTask := func(threadId int) error {
				var err error
				logMsgPrefix := clientutils.GetLogMsgPrefix(threadId, ps.IsDryRun())

				storageAPI := path.Join("api", "storage", relativePath)
				setPropertiesURL, err := clientutils.BuildUrl(ps.GetArtifactoryDetails().GetUrl(), storageAPI, make(map[string]string))
				if err != nil {
					return err
				}
				// Because we do set/delete props on search results that took into account the
				// recursive flag, we do not want the action itself to be recursive.
				setPropertiesURL += "?properties=" + encodedParam + "&recursive=0"
				resp, body, err := action(logMsgPrefix, relativePath, setPropertiesURL)

				if err != nil {
					return err
				}
				if err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusNoContent); err != nil {
					return err
				}
				successCounters[threadId]++
				return nil
			}

			_, _ = producerConsumer.AddTaskWithError(setPropsTask, errorsQueue.AddError)
		}
		defer producerConsumer.Done()
		if err := reader.GetError(); err != nil {
			errorsQueue.AddError(err)
		}
		reader.Reset()
	}()

	producerConsumer.Run()
	totalSuccess := 0
	for _, v := range successCounters {
		totalSuccess += v
	}
	return totalSuccess, errorsQueue.GetError()
}

func (ps *PropsService) sendDeleteRequest(logMsgPrefix, relativePath, setPropertiesUrl string) (resp *http.Response, body []byte, err error) {
	log.Info(logMsgPrefix+"Deleting properties on:", relativePath)
	httpClientsDetails := ps.GetArtifactoryDetails().CreateHttpClientDetails()
	resp, body, err = ps.client.SendDelete(setPropertiesUrl, nil, &httpClientsDetails)
	return
}

func (ps *PropsService) sendPutRequest(logMsgPrefix, relativePath, setPropertiesUrl string) (resp *http.Response, body []byte, err error) {
	log.Info(logMsgPrefix+"Setting properties on:", relativePath)
	httpClientsDetails := ps.GetArtifactoryDetails().CreateHttpClientDetails()
	resp, body, err = ps.client.SendPut(setPropertiesUrl, nil, &httpClientsDetails)
	return
}

func NewPropsParams() PropsParams {
	return PropsParams{}
}

func (ps *PropsService) GetItemProperties(relativePath string) (*utils.ItemProperties, error) {
	restAPI := path.Join("api", "storage", path.Clean(relativePath))
	propertiesURL, err := clientutils.BuildUrl(ps.GetArtifactoryDetails().GetUrl(), restAPI, make(map[string]string))
	if err != nil {
		return nil, err
	}
	propertiesURL += "?properties"

	httpClientsDetails := ps.GetArtifactoryDetails().CreateHttpClientDetails()
	resp, body, _, err := ps.client.SendGet(propertiesURL, true, &httpClientsDetails)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound && strings.Contains(string(body), "No properties could be found") {
		return nil, nil
	}
	if err = errorutils.CheckResponseStatusWithBody(resp, body, http.StatusOK); err != nil {
		return nil, err
	}
	log.Debug("Artifactory response:", resp.Status)

	result := &utils.ItemProperties{}
	err = json.Unmarshal(body, result)
	return result, errorutils.CheckError(err)
}
