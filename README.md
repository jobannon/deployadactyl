# Deployadactyl

Deployadactyl is a Go client library for deploying applications to multiple [Cloud Foundry](https://www.cloudfoundry.org/) instances. If a deployment fails in any instance, it will automatically rollback.

Deployadactyl requires Go version 1.6 or greater.

**Documentation:** _godoc link/badge_
**Build Status:** _build status badge_
**Test Coverage:** _coverage badge_

With Deployadactyl you can register your event handlers to perform any additional actions your deployment flow may require. For us, this meant adding handlers that would open and close change records, as well as notify anyone on pager duty of significant events.

### Dependencies

Deployadactyl has the following dependencies:

|Name|Reason|
|---|---|
|[Gin Web Framework](https://github.com/gin-gonic/gin)|Used as our server.|
|[Go-errors](https://github.com/go-errors/errors)|Better errors with stacktraces.|
|[Golang logging library](https://github.com/op/go-logging)|Easily managable logging with logging levels.|

## Usage

```go
import "github.com/compozed/deployadactyl/creator"
```

Deployadactyl needs a configuration `yaml` file and a logging level in order to run. The logging level needs to be of type `logging.LogLevel`. These values should be passed into the Creator.

After creating the Creator, you *can* create a default logger off of it that will format your log messages to match Deployadactyl's log format. An example has been provided below.



### Simple example usage

```go
package main

import (
  "net/http"
  "os"

	"github.com/me/deployadactyl-consumer/mypackager"
  "github.com/compozed/deployadactyl/creator"
  "github.com/op/go-logging"
)

const (
  defaultConfig = "./config.yml"
  defaultLevel  = "DEBUG"
)

func main() {
	logLevel, _ := logging.LogLevel(defaultLevel)
	log := logger.DefaultLogger(os.Stdout, logLevel, "deployadactyl-consumer")

	c, err := creator.New(defaultLevel, defaultConfig)
	if err != nil {
		log.Fatal(err)
	}

    // Creating this logger is optional
	l := c.CreateLogger()

	listener := c.CreateListener()
	l.Infof("Listening on Port %d", c.CreateConfig().Port)

	em := c.CreateEventManager()

	myPackageHandler := mypackager.CreateMyPackageHandler()
	em.AddHandler(myPackageHandler, "deploy.start")
	em.AddHandler(myPackageHandler, "deploy.finish")
	em.AddHandler(myPackageHandler, "deploy.error")

	hf := c.CreateHandlerFunc()

	err = http.Serve(listener, hf)
	if err != nil {
		l.Fatal(err)
	}
}
}
```

### Available logging levels

### Example configuration yaml file
```
---
environments:
  - name: my-env-1
    domain: my-env-1.example.com
		extra: value
    foundations:
    - https://my-env-1.foundation-1.example.com
    - https://my-env-1.foundation-2.example.com

  - name: preproduction
    domain: app1.cftest.allstate.com
    allow_page: false
    authenticate: false
    foundations:
    - https://api.sd1.cftest.allstate.com

  - name: prod-dmz
    domain: cws.allstate.com
    allow_page: true
    authenticate: true
    foundations:
    - https://api.cf.prod-dmz.ro1.allstate.com
    - https://api.cf.prod-dmz.ro2.allstate.com
    - https://api.cf.prod-dmz.gl1.allstate.com
    - https://api.cf.prod-dmz.gl2.allstate.com

  - name: prod-mpn
    domain: platform.allstate.com
    allow_page: true
    authenticate: true
    foundations:
    - https://api.cf.prod-mpn.ro1.allstate.com
    - https://api.cf.prod-mpn.ro2.allstate.com
    - https://api.cf.prod-mpn.gl1.allstate.com
    - https://api.cf.prod-mpn.gl2.allstate.com
```

### Available emitted events
- "deploy.start"
- "deploy.finish"
- "deploy.error"
- "validate.foundationsUnavailable"

## How to push your app

## How it works

Deployadactyl works by utilizing the [Cloud Foundry CLI](http://docs.cloudfoundry.org/cf-cli/) to push your application. It grabs a list of foundations from the Deployadactyl config, logs into each one and calls `cf push`. The general flow is to fetch your artifact, unzip it, and push it. Deployadactyl utilizes [blue green deployments](https://docs.pivotal.io/pivotalcf/devguide/deploy-apps/blue-green.html) and if it's unable to push your application it will rollback to the previous version.

## Contributing

- Fork the project
- Make a branch
- Commit to the branch
- Send us a Pull Request
