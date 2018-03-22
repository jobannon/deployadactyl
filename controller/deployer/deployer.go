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

	"encoding/base64"
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

func (d SilentDeployer) Deploy(authorization I.Authorization, body io.Reader, actionCreator I.ActionCreator, environment, org, space, appName, uuid string, contentType I.DeploymentType, response io.ReadWriter) *I.DeployResponse {
	url := os.Getenv("SILENT_DEPLOY_URL")
	deployResponse := &I.DeployResponse{}

	request, err := http.NewRequest("POST", fmt.Sprintf(url+"/%s/%s/%s", org, space, appName), body)
	if err != nil {
		log.Println(fmt.Sprintf("Silent deployer request err: %s", err))
		deployResponse.Error = err
	}
	usernamePassword := base64.StdEncoding.EncodeToString([]byte(authorization.Username + ":" + authorization.Password))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", usernamePassword)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Do(request)
	if err != nil {
		log.Println(fmt.Sprintf("Silent deployer response err: %s", err))
		deployResponse.StatusCode = resp.StatusCode
		deployResponse.Error = err
	}

	deployResponse.StatusCode = resp.StatusCode
	deployResponse.Error = err
	return deployResponse
}

type Deployer struct {
	Config       config.Config
	BlueGreener  I.BlueGreener
	Prechecker   I.Prechecker
	EventManager I.EventManager
	Randomizer   I.Randomizer
	ErrorFinder  I.ErrorFinder
	Log          I.Logger
	FileSystem   *afero.Afero
}

func (d Deployer) Deploy(authorization I.Authorization, body io.Reader, actionCreator I.ActionCreator, environment, org, space, appName, uuid string, contentType I.DeploymentType, response io.ReadWriter) *I.DeployResponse {
	var (
		environments           = d.Config.Environments
		authenticationRequired = environments[environment].Authenticate
		manifest               string
		appPath                string
		instances              uint16
	)
	if uuid == "" {
		uuid = d.Randomizer.StringRunes(10)
	}
	deploymentLogger := logger.DeploymentLogger{d.Log, uuid}

	deploymentInfo := &S.DeploymentInfo{}
	deploymentInfo.Org = org
	deploymentInfo.Space = space
	deploymentInfo.AppName = appName
	deploymentInfo.UUID = uuid
	deploymentInfo.Environment = environment
	deploymentInfo.CustomParams = make(map[string]interface{})

	deployResponse := &I.DeployResponse{
		DeploymentInfo: deploymentInfo,
	}

	e, ok := environments[environment]
	if !ok {
		fmt.Fprintln(response, EnvironmentNotFoundError{environment}.Error())
		deployResponse.Error = EnvironmentNotFoundError{environment}
		deployResponse.StatusCode = http.StatusInternalServerError
		return deployResponse
	}
	d.Log.Debugf("Starting deploy of %s with UUID %s", appName, uuid)

	deploymentInfo.SkipSSL = environments[environment].SkipSSL
	deploymentInfo.Domain = environments[environment].Domain

	deploymentInfo.CustomParams = environments[environment].CustomParams

	deploymentLogger.Debug("building deploymentInfo")

	if contentType.JSON {
		deploymentInfo, err := getDeploymentInfo(body, deploymentInfo)
		if err != nil {
			deploymentLogger.Error(err)
			deployResponse.StatusCode = http.StatusInternalServerError
			deployResponse.Error = err
			deployResponse.DeploymentInfo = deploymentInfo
			return deployResponse
		}
	}

	deployEventData := &S.DeployEventData{Response: response, DeploymentInfo: deploymentInfo, RequestBody: body}

	defer emitDeployFinish(d, deployEventData, response, deployResponse, deploymentLogger)
	defer emitDeploySuccess(d, deployEventData, response, deployResponse, deploymentLogger)

	deploymentLogger.Debugf("emitting a %s event", C.DeployStartEvent)
	err := d.EventManager.Emit(I.Event{Type: C.DeployStartEvent, Data: deployEventData})
	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.InitializationError{err}
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = EventError{Type: C.DeployStartEvent, Err: err}
		return deployResponse
	}

	defer func() { d.FileSystem.RemoveAll(appPath) }()
	d.Log.Debugf("Starting deploy of %s with UUID %s", appName, uuid)

	deploymentLogger.Debug("prechecking the foundations")
	err = d.Prechecker.AssertAllFoundationsUp(environments[environment])
	if err != nil {
		deploymentLogger.Error(err)
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = err
		return deployResponse
	}

	deploymentLogger.Debug("checking for basic auth")
	//username, password, ok := req.BasicAuth()
	if authorization.Username == "" && authorization.Password == "" {
		if authenticationRequired {
			deployResponse.StatusCode = http.StatusUnauthorized
			deployResponse.Error = BasicAuthError{}
			return deployResponse
		}
		authorization.Username = d.Config.Username
		authorization.Password = d.Config.Password
	}

	if contentType.JSON {
		deploymentLogger.Debug("deploying from json request")

		deploymentInfo.ContentType = "JSON"
	} else if contentType.ZIP {
		deploymentLogger.Debug("deploying from zip request")
		deploymentInfo.Body = body
		deploymentInfo.ContentType = "ZIP"
	} else {
		deployResponse.StatusCode = http.StatusBadRequest
		deployResponse.Error = InvalidContentTypeError{}
		return deployResponse
	}

	deploymentLogger.Debugf("emitting a %s event", C.ArtifactRetrievalStart)
	err = d.EventManager.Emit(I.Event{Type: C.ArtifactRetrievalStart, Data: deployEventData})

	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.InitializationError{err}
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = EventError{Type: C.ArtifactRetrievalStart, Err: err}
		return deployResponse
	}

	appPath, manifest, instances, err = actionCreator.SetUp(*deploymentInfo, environments[environment].Instances)

	if err != nil {
		deploymentLogger.Error(err)
		_ = d.EventManager.Emit(I.Event{Type: C.ArtifactRetrievalFailure, Data: deployEventData})
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = err
		return deployResponse
	}
	deploymentLogger.Debugf("emitting a %s event", C.ArtifactRetrievalSuccess)

	err = d.EventManager.Emit(I.Event{Type: C.ArtifactRetrievalSuccess, Data: deployEventData})
	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.InitializationError{err}
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = EventError{Type: C.ArtifactRetrievalSuccess, Err: err}
		return deployResponse
	}

	deploymentInfo.Username = authorization.Username
	deploymentInfo.Password = authorization.Password
	deploymentInfo.Manifest = manifest
	deploymentInfo.AppPath = appPath
	deploymentInfo.Instances = instances

	defer func() { d.FileSystem.RemoveAll(deploymentInfo.AppPath) }()

	deploymentMessage := fmt.Sprintf(deploymentOutput, deploymentInfo.ArtifactURL, deploymentInfo.Username, deploymentInfo.Environment, deploymentInfo.Org, deploymentInfo.Space, deploymentInfo.AppName)
	deploymentLogger.Info(deploymentMessage)
	fmt.Fprintln(response, deploymentMessage)

	enableRollback := e.EnableRollback
	err = d.EventManager.Emit(I.Event{Type: C.PushStartedEvent, Data: deployEventData})
	if err != nil {
		deploymentLogger.Error(err)
		err = &bluegreen.InitializationError{err}
		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = EventError{Type: C.PushStartedEvent, Err: err}
		return deployResponse
	}

	err = d.BlueGreener.Execute(actionCreator, e, appPath, *deploymentInfo, response)

	if err != nil {
		if !enableRollback {
			deploymentLogger.Errorf("EnableRollback %t, returning status %d and err %s", enableRollback, http.StatusOK, err)
			deployResponse.StatusCode = http.StatusOK
			deployResponse.Error = err
			return deployResponse
		}

		if matched, _ := regexp.MatchString("login failed", err.Error()); matched {
			deployResponse.StatusCode = http.StatusBadRequest
			deployResponse.Error = err
			return deployResponse
		}

		deployResponse.StatusCode = http.StatusInternalServerError
		deployResponse.Error = err
		return deployResponse
	}

	deploymentLogger.Infof("successfully deployed application %s", deploymentInfo.AppName)
	fmt.Fprintf(response, "\n%s", successfulDeploy)

	deployResponse.StatusCode = http.StatusOK
	deployResponse.Error = err
	return deployResponse
}

func getDeploymentInfo(reader io.Reader, deploymentInfo *S.DeploymentInfo) (*S.DeploymentInfo, error) {
	err := json.NewDecoder(reader).Decode(deploymentInfo)
	if err != nil {
		return deploymentInfo, err
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
	return deploymentInfo, nil
}

func emitDeployFinish(d Deployer, deployEventData *S.DeployEventData, response io.ReadWriter, deployResponse *I.DeployResponse, deploymentLogger logger.DeploymentLogger) {
	deploymentLogger.Debugf("emitting a %s event", C.DeployFinishEvent)
	finishErr := d.EventManager.Emit(I.Event{Type: C.DeployFinishEvent, Data: deployEventData})
	if finishErr != nil {
		fmt.Fprintln(response, finishErr)
		err := bluegreen.FinishDeployError{Err: fmt.Errorf("%s: %s", deployResponse.Error, EventError{C.DeployFinishEvent, finishErr})}
		deployResponse.Error = err
		deployResponse.StatusCode = http.StatusInternalServerError
	}
}

func emitDeploySuccess(d Deployer, deployEventData *S.DeployEventData, response io.ReadWriter, deployResponse *I.DeployResponse, deploymentLogger logger.DeploymentLogger) {
	deployEvent := I.Event{Type: C.DeploySuccessEvent, Data: deployEventData}
	if deployResponse.Error != nil {
		printErrors(d, response, &deployResponse.Error)

		deployEvent.Type = C.DeployFailureEvent
		deployEvent.Error = deployResponse.Error
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
