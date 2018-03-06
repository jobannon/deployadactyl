package mocks

//import (
//	"fmt"
//	"github.com/compozed/deployadactyl/interfaces"
//	"io"
//)
//
//type StateManager struct {
//	DeployCall struct {
//		Called   int
//		Received struct {
//			Request     *http.Request
//			Environment string
//			Org         string
//			Space       string
//			AppName     string
//			UUID        string
//			ContentType I.DeploymentType
//			Response    io.ReadWriter
//		}
//		Write struct {
//			Output string
//		}
//		Returns struct {
//			Error      error
//			StatusCode int
//		}
//	}
//}
//
//// Deploy mock method.
//func (s *StateManager) Stop(req *http.Request, environment, org, space, appName, uuid string, contentType I.DeploymentType, out io.ReadWriter, reqChan chan I.DeployResponse) {
//	s.DeployCall.Called++
//
//	s.DeployCall.Received.Request = req
//	s.DeployCall.Received.Environment = environment
//	s.DeployCall.Received.Org = org
//	s.DeployCall.Received.Space = space
//	s.DeployCall.Received.AppName = appName
//	s.DeployCall.Received.UUID = uuid
//	s.DeployCall.Received.ContentType = contentType
//	s.DeployCall.Received.Response = out
//
//	fmt.Fprint(out, s.DeployCall.Write.Output)
//
//	response := interfaces.StartStopEventData{
//		StatusCode: s.DeployCall.Returns.StatusCode,
//		Error:      s.DeployCall.Returns.Error,
//	}
//
//	reqChan <- response
//}
