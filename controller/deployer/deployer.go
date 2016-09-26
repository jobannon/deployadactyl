// Package deployer will deploy your application.
package deployer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/compozed/deployadactyl/config"
	"github.com/compozed/deployadactyl/controller/deployer/manifestro"
	"github.com/compozed/deployadactyl/geterrors"
	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
	"github.com/spf13/afero"
)

const (
	basicAuthHeaderNotFound   = "basic auth header not found"
	environmentNotFound       = "environment not found"
	cannotFetchArtifact       = "cannot fetch artifact"
	invalidArtifact           = "invalid artifact"
	invalidPostRequest        = "invalid POST request"
	cannotOpenManifestFile    = "cannot open manifest file"
	cannotFindManifestFile    = "cannot find manifest file in zip"
	cannotPrintToManifestFile = "cannot print to open manifest file"
	successfulDeploy          = `Your deploy was successful! (^_^)d
If you experience any problems after this point, check that you can manually push your application to Cloud Foundry on a lower environment.
It is likely that it is an error with your application and not with Deployadactyl.
Thanks for using Deployadactyl! Please push down pull up on your lap bar and exit to your left.`
	deployStartError  = "an error occurred in the deploy.start event"
	deployFinishError = "an error occurred in the deploy.finish event"
	deploymentOutput  = `Deployment Parameters:
	Artifact URL: %s,
	Username:     %s,
	Environment:  %s,
	Org:          %s,
	Space:        %s,
	AppName:      %s`
)

// Deployer contains the bluegreener for deployments, environment variables, a fetcher for artifacts, a prechecker and event manager.
type Deployer struct {
	Config       config.Config
	BlueGreener  I.BlueGreener
	Fetcher      I.Fetcher
	Prechecker   I.Prechecker
	EventManager I.EventManager
	Randomizer   I.Randomizer
	Log          *logging.Logger
	FileSystem   *afero.Afero
}

// Deploy takes the deployment information, checks the foundations, fetches the artifact and deploys the application.
func (d Deployer) Deploy(req *http.Request, environmentName, org, space, appName, contentType string, out io.Writer) (err error, statusCode int) {
	var (
		deploymentInfo         = S.DeploymentInfo{}
		environments           = d.Config.Environments
		authenticationRequired = environments[environmentName].Authenticate
		deployEventData        = S.DeployEventData{}
		manifest               []byte
		appPath                string
	)

	err = d.Prechecker.AssertAllFoundationsUp(environments[environmentName])
	if err != nil {
		fmt.Fprintln(out, err)
		return errors.New(err), http.StatusInternalServerError
	}

	// move into controller
	username, password, ok := req.BasicAuth()
	if !ok {
		if authenticationRequired {
			return errors.New(basicAuthHeaderNotFound), http.StatusUnauthorized
		}
		username = d.Config.Username
		password = d.Config.Password
	}

	if isJSON(contentType) {
		deploymentInfo, err = getDeploymentInfo(req.Body)
		if err != nil {
			fmt.Fprintln(out, err)
			return err, http.StatusInternalServerError
		}

		if deploymentInfo.Manifest != "" {
			manifest, err = base64.StdEncoding.DecodeString(deploymentInfo.Manifest)
			if err != nil {
				fmt.Fprintln(out, err)
				return errors.New(cannotOpenManifestFile), http.StatusBadRequest
			}
		}

		appPath, err = d.Fetcher.Fetch(deploymentInfo.ArtifactURL, deploymentInfo.Manifest)
		if err != nil {
			fmt.Fprintln(out, err)
			return err, http.StatusInternalServerError
		}
		defer d.FileSystem.RemoveAll(appPath)

	} else if isZip(contentType) {
		appPath, err = d.Fetcher.FetchFromZip(req)
		defer d.FileSystem.RemoveAll(appPath)
		if err != nil {
			return err, http.StatusInternalServerError
		}

		manifest, err = d.FileSystem.ReadFile(appPath + "/manifest.yml")
		if err != nil {
			fmt.Fprintln(out, cannotFindManifestFile)
		}

		deploymentInfo.ArtifactURL = "Local Developer App Deploy " + appPath
	} else {
		return errors.New("must be application/json or application/zip"), http.StatusBadRequest
	}

	deploymentInfo.Username = username
	deploymentInfo.Password = password
	deploymentInfo.Environment = environmentName
	deploymentInfo.Org = org
	deploymentInfo.Space = space
	deploymentInfo.AppName = appName
	deploymentInfo.UUID = d.Randomizer.StringRunes(128)
	deploymentInfo.SkipSSL = environments[environmentName].SkipSSL
	deploymentInfo.Instances = 1
	deploymentInfo.Manifest = string(manifest)

	deploymentMessage := fmt.Sprintf(deploymentOutput, deploymentInfo.ArtifactURL, deploymentInfo.Username, deploymentInfo.Environment, deploymentInfo.Org, deploymentInfo.Space, deploymentInfo.AppName)
	d.Log.Debug(deploymentMessage)
	fmt.Fprintln(out, deploymentMessage)

	deployEventData = S.DeployEventData{
		Writer:         out,
		DeploymentInfo: &deploymentInfo,
		RequestBody:    req.Body,
	}

	defer func() (error, int) {
		deployFinishEvent := S.Event{
			Type: "deploy.finish",
			Data: deployEventData,
		}

		eventErr := d.EventManager.Emit(deployFinishEvent)
		if eventErr != nil {
			fmt.Fprintln(out, eventErr)
		}

		if err != nil {
			return err, statusCode
		}

		return nil, http.StatusOK
	}()

	deployStartEvent := S.Event{
		Type: "deploy.start",
		Data: deployEventData,
	}

	err = d.EventManager.Emit(deployStartEvent)
	if err != nil {
		fmt.Fprintln(out, err)
		return errors.New(deployStartError), http.StatusInternalServerError
	}

	deployEventData = S.DeployEventData{
		Writer:         out,
		DeploymentInfo: &deploymentInfo,
	}

	environment, found := environments[deploymentInfo.Environment]
	if !found {
		var deployEvent = S.Event{
			Type: "deploy.error",
			Data: deployEventData,
		}

		err = d.EventManager.Emit(deployEvent)
		if err != nil {
			fmt.Fprintln(out, err)
		}

		err = errors.Errorf("%s: %s", environmentNotFound, deploymentInfo.Environment)
		fmt.Fprintln(out, err)
		return err, http.StatusInternalServerError
	}

	instances := manifestro.GetInstances(deploymentInfo.Manifest)
	if instances != nil {
		deploymentInfo.Instances = *instances
	} else {
		deploymentInfo.Instances = environments[environmentName].Instances
	}

	defer func() {
		var deployEvent = S.Event{
			Type: "deploy.success",
			Data: deployEventData,
		}

		if err != nil {
			deployEvent.Type = "deploy.failure"
		}

		eventErr := d.EventManager.Emit(deployEvent)
		if eventErr != nil {
			fmt.Fprintln(out, eventErr)
		}
	}()

	err = d.BlueGreener.Push(environment, appPath, deploymentInfo, out)
	if err != nil {
		if matched, _ := regexp.MatchString("login failed", err.Error()); matched {
			return err, http.StatusBadRequest
		}
		return err, http.StatusInternalServerError
	}

	fmt.Fprintln(out, fmt.Sprintf("\n%s", successfulDeploy))
	return err, http.StatusOK
}

func getDeploymentInfo(reader io.Reader) (S.DeploymentInfo, error) {
	deploymentInfo := S.DeploymentInfo{}
	err := json.NewDecoder(reader).Decode(&deploymentInfo)
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
		return S.DeploymentInfo{}, err
	}
	return deploymentInfo, nil
}

func isZip(contentType string) bool {
	return contentType == "application/zip"
}

func isJSON(contentType string) bool {
	return contentType == "application/json"
}
