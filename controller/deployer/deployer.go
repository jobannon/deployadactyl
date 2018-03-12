// Package deployer will deploy your application.
package deployer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"bytes"

	"crypto/tls"
	"log"
	"os"

	"github.com/compozed/deployadactyl/config"
	C "github.com/compozed/deployadactyl/constants"
	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/geterrors"
	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/spf13/afero"
)

const (
	successfulDeploy = `Your deploy was successful! (^_^)b
If you experience any problems after this point, check that you can manually push your application to Cloud Foundry on a lower environment.
It is likely that it is an error with your application and not with Deployadactyl.
Thanks for using Deployadactyl! Please push down pull up on your lap bar and exit to your left.

`

	deploymentOutput = `Deployment Parameters:
Artifact URL: %s,
Username:     %s,
Environment:  %s,
Org:          %s,
Space:        %s,
AppName:      %s`
)

type SilentDeployer struct {
}

func (d SilentDeployer) Deploy(req *http.Request, environment, org, space, appName, uuid string, contentType I.DeploymentType, response io.ReadWriter, reqChannel chan I.DeployResponse) {
	url := os.Getenv("SILENT_DEPLOY_URL")
	deployResponse := I.DeployResponse{}

	request, err := http.NewRequest("POST", fmt.Sprintf(url+"/%s/%s/%s", org, space, appName), req.Body)
	if err != nil {
		log.Println(fmt.Sprintf("Silent deployer request err: %s", err))
		deployResponse.Error = err
		reqChannel <- deployResponse
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", req.Header.Get("Authorization"))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Do(request)
	if err != nil {
		log.Println(fmt.Sprintf("Silent deployer response err: %s", err))
		deployResponse.StatusCode = resp.StatusCode
		deployResponse.Error = err
		reqChannel <- deployResponse
	}

	deployResponse.StatusCode = resp.StatusCode
	deployResponse.Error = err
	reqChannel <- deployResponse
}

type Deployer struct {
	Config         config.Config
	BlueGreener    I.BlueGreener
	PusherCreator  I.ActionCreator
	StopperCreator I.ActionCreator
	Fetcher        I.Fetcher
	Prechecker     I.Prechecker
	EventManager   I.EventManager
	Randomizer     I.Randomizer
	ErrorFinder    I.ErrorFinder
	Log            I.Logger
	FileSystem     *afero.Afero
}

func (d Deployer) Deploy(req *http.Request, environment, org, space, appName, uuid string, contentType I.DeploymentType, response io.ReadWriter, reqChannel chan I.DeployResponse) {
	deployResponse := I.DeployResponse{}
	statusCode, deploymentInfo, err := d.deployInternal(
		req,
		environment,
		org,
		space,
		appName,
		uuid,
		contentType,
		response,
	)
	deployResponse.StatusCode = statusCode
	deployResponse.DeploymentInfo = deploymentInfo
	deployResponse.Error = err
	reqChannel <- deployResponse
}

func (d Deployer) deployInternal(req *http.Request, environment, org, space, appName, uuid string, contentType I.DeploymentType, response io.ReadWriter) (statusCode int, deploymentInfo *S.DeploymentInfo, err error) {
	var (
		environments           = d.Config.Environments
		authenticationRequired = environments[environment].Authenticate
		deployEventData        = S.DeployEventData{}
		appPath                string
	)
	deploymentInfo = &S.DeploymentInfo{}

	if uuid == "" {
		uuid = d.Randomizer.StringRunes(10)
	}

	defer func() { d.FileSystem.RemoveAll(appPath) }()
	d.Log.Debugf("Starting deploy of %s with UUID %s", appName, uuid)
	deploymentLogger := logger.DeploymentLogger{d.Log, uuid}

	e, ok := environments[environment]
	if !ok {
		fmt.Fprintln(response, EnvironmentNotFoundError{environment}.Error())
		return http.StatusInternalServerError, deploymentInfo, EnvironmentNotFoundError{environment}
	}

	deploymentLogger.Debug("prechecking the foundations")
	err = d.Prechecker.AssertAllFoundationsUp(environments[environment])
	if err != nil {
		deploymentLogger.Error(err)
		return http.StatusInternalServerError, deploymentInfo, err
	}

	deploymentLogger.Debug("checking for basic auth")
	username, password, ok := req.BasicAuth()
	if !ok {
		if authenticationRequired {
			return http.StatusUnauthorized, deploymentInfo, BasicAuthError{}
		}
		username = d.Config.Username
		password = d.Config.Password
	}

	deploymentLogger.Debug("deploying from json request")
	deploymentLogger.Debug("building deploymentInfo")
	deploymentInfo, err = getDeploymentInfo(req.Body)
	if err != nil {
		deploymentLogger.Error(err)
		return http.StatusInternalServerError, deploymentInfo, err
	}

	if contentType.JSON {

	} else if contentType.ZIP {
		deploymentLogger.Debug("deploying from zip request")
		appPath, err = d.Fetcher.FetchZipFromRequest(req)
		if err != nil {
			return http.StatusInternalServerError, deploymentInfo, err
		}

		//manifest, _ := d.FileSystem.ReadFile(appPath + "/manifest.yml")

		deploymentInfo.ArtifactURL = appPath
	} else {
		return http.StatusBadRequest, deploymentInfo, InvalidContentTypeError{}
	}

	deploymentInfo.Username = username
	deploymentInfo.Password = password
	deploymentInfo.Environment = environment
	deploymentInfo.Org = org
	deploymentInfo.Space = space
	deploymentInfo.AppName = appName
	deploymentInfo.UUID = uuid
	deploymentInfo.SkipSSL = environments[environment].SkipSSL
	deploymentInfo.Domain = environments[environment].Domain
	deploymentInfo.AppPath = appPath
	deploymentInfo.CustomParams = make(map[string]interface{})
	deploymentInfo.CustomParams = environments[environment].CustomParams
	deploymentInfo.Instances = environments[environment].Instances

	// TODO This next block looks like dead code as the check already occurred at the beginning of the function
	e, found := environments[deploymentInfo.Environment]
	if !found {
		err = d.EventManager.Emit(I.Event{Type: C.DeployErrorEvent, Data: deployEventData})
		if err != nil {
			deploymentLogger.Error(err)
		}

		err = fmt.Errorf("environment not found: %s", deploymentInfo.Environment)
		deploymentLogger.Error(err)
		return http.StatusInternalServerError, deploymentInfo, err
	}

	deploymentMessage := fmt.Sprintf(deploymentOutput, deploymentInfo.ArtifactURL, deploymentInfo.Username, deploymentInfo.Environment, deploymentInfo.Org, deploymentInfo.Space, deploymentInfo.AppName)
	deploymentLogger.Info(deploymentMessage)
	fmt.Fprintln(response, deploymentMessage)

	deployEventData = S.DeployEventData{Response: response, DeploymentInfo: deploymentInfo, RequestBody: req.Body}

	defer emitDeployFinish(d, deployEventData, response, &err, &statusCode, deploymentLogger)
	defer emitDeploySuccess(d, deployEventData, response, &err, &statusCode, deploymentLogger)

	deploymentLogger.Debugf("emitting a %s event", C.DeployStartEvent)
	err = d.EventManager.Emit(I.Event{Type: C.DeployStartEvent, Data: deployEventData})
	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.InitializationError{err}
		return http.StatusInternalServerError, deploymentInfo, EventError{Type: C.DeployStartEvent, Err: err}
	}

	enableRollback := e.EnableRollback

	err = d.BlueGreener.Execute(d.PusherCreator, e, appPath, *deploymentInfo, response)

	if err != nil {
		if !enableRollback {
			deploymentLogger.Errorf("EnableRollback %t, returning status %d and err %s", enableRollback, http.StatusOK, err)
			return http.StatusOK, deploymentInfo, err
		}

		if matched, _ := regexp.MatchString("login failed", err.Error()); matched {
			return http.StatusBadRequest, deploymentInfo, err
		}

		return http.StatusInternalServerError, deploymentInfo, err
	}

	deploymentLogger.Infof("successfully deployed application %s", deploymentInfo.AppName)
	fmt.Fprintf(response, "\n%s", successfulDeploy)

	return http.StatusOK, deploymentInfo, err
}

func getDeploymentInfo(reader io.Reader) (*S.DeploymentInfo, error) {
	deploymentInfo := S.DeploymentInfo{}
	err := json.NewDecoder(reader).Decode(&deploymentInfo)
	if err != nil {
		return &deploymentInfo, err
	}

	getter := geterrors.WrapFunc(func(key string) string {
		if key == "artifact_url" {
			return deploymentInfo.ArtifactURL
		}
		return ""
	})

	getter.Get("artifact_url")

	err = getter.Err("The following properties are missing")
	if err != nil {
		return &S.DeploymentInfo{}, err
	}
	return &deploymentInfo, nil
}

func emitDeployFinish(d Deployer, deployEventData S.DeployEventData, response io.ReadWriter, err *error, statusCode *int, deploymentLogger logger.DeploymentLogger) {
	deploymentLogger.Debugf("emitting a %s event", C.DeployFinishEvent)

	finishErr := d.EventManager.Emit(I.Event{Type: C.DeployFinishEvent, Data: deployEventData})
	if finishErr != nil {
		fmt.Fprintln(response, finishErr)
		*err = bluegreen.FinishDeployError{Err: fmt.Errorf("%s: %s", *err, EventError{C.DeployFinishEvent, finishErr})}
		*statusCode = http.StatusInternalServerError
	}
}

func emitDeploySuccess(d Deployer, deployEventData S.DeployEventData, response io.ReadWriter, err *error, statusCode *int, deploymentLogger logger.DeploymentLogger) {
	deployEvent := I.Event{Type: C.DeploySuccessEvent, Data: deployEventData}
	if *err != nil {
		printErrors(d, response, err)

		deployEvent.Type = C.DeployFailureEvent
		deployEvent.Error = *err
	}

	deploymentLogger.Debug(fmt.Sprintf("emitting a %s event", deployEvent.Type))
	eventErr := d.EventManager.Emit(deployEvent)
	if eventErr != nil {
		deploymentLogger.Errorf("an error occurred when emitting a %s event: %s", deployEvent.Type, eventErr)
		fmt.Fprintln(response, eventErr)
	}
}

func printErrors(d Deployer, response io.ReadWriter, err *error) {
	tempBuffer := bytes.Buffer{}
	tempBuffer.ReadFrom(response)
	fmt.Fprint(response, tempBuffer.String())

	errors := d.ErrorFinder.FindErrors(tempBuffer.String())
	if len(errors) > 0 {
		*err = errors[0]
		for _, error := range errors {
			fmt.Fprintln(response)
			fmt.Fprintln(response, "*******************")
			fmt.Fprintln(response)
			fmt.Fprintln(response, "The following error was found in the above logs: "+error.Error())
			fmt.Fprintln(response)
			fmt.Fprintln(response, "Error: "+error.Details()[0])
			fmt.Fprintln(response)
			fmt.Fprintln(response, "Potential solution: "+error.Solution())
			fmt.Fprintln(response)
			fmt.Fprintln(response, "*******************")
		}
	}
}
