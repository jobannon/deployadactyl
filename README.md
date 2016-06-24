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

  "github.com/compozed/deployadactyl/creator"
  "github.com/op/go-logging"
)

const (
  defaultConfig = "./config.yml"
  defaultLevel  = "DEBUG"
)

func main() {
  // Create a temporary logger until we can create the Creator
  logLevel, _ := logging.LogLevel(defaultLevel)
  log := logger.DefaultLogger(os.Stdout, logLevel, "deployadactyl-consumer")

  c, err := creator.New(defaultLevel, defaultConfig)
  if err != nil {
    log.Fatal(err)
  }

  // This is an optional logger that makes logs look nice
  l := c.CreateLogger()

  listener := c.CreateListener()
  l.Infof("Listening on Port %d", c.CreateConfig().Port)

  dh := c.CreateDeployadactylHandler()

  err = http.Serve(listener, dh)
  if err != nil {
    l.Fatal(err)
  }
}
```

### Available logging levels

### Configuration File

The config file is used to specify your environments.

You can add in extra miscellaneous information here, and it will be added to the `Config` struct which can be accessed via `Creator.cfg`. This is useful because you can maintain one config file and still access configuration items in your event handlers.

#### Example configuration yaml file
```yaml
---
environments:
  - name: my-env-1
    domain: my-env-1.example.com
    some_extra: value
    foundations:
    - https://my-env-1.foundation-1.example.com
    - https://my-env-1.foundation-2.example.com

  - name: my-env-2
    domain: my-env-2.example.com
    some_extra: value
    foundations:
    - https://my-env-2.foundation-1.example.com
    - https://my-env-2.foundation-2.example.com

  - name: my-env-3
    domain: my-env-3.example.com
    some_extra: value
    foundations:
    - https://my-env-3.foundation-1.example.com
    - https://my-env-3.foundation-2.example.com

  - name: my-env-4
    domain: my-env-4.example.com
    some_extra: value
    foundations:
    - https://my-env-4.foundation-1.example.com
    - https://my-env-4.foundation-2.example.com
```
### Event handling

There are a number of events available for you to register handlers to.

Events will provide an `Event` struct:

```go
type Event struct {
	Type string
	Data interface{}
}
```

The `Type` string will contain the type of event that it is. Depending on the type of event `Data` interface will contain different kinds of data.

#### Available emitted event types

<table>
<thead>
  <tr>
    <td><strong>Event Type</strong></td>
    <td><strong>Data</strong></td>
  </tr>
</thead>
<tbody>
  <tr>
    <td>
      <p><code>deploy.start</code></p>
      <p><code>deploy.finish</code></p>
      <p><code>deploy.error</code></p>
    </td>
    <td>
<div class="highlight highlight-source-go"><pre>
type DeployEventData struct {
	Writer         io.Writer
	DeploymentInfo *DeploymentInfo
	RequestBody    io.Reader
}
</pre></div>
    </td>
  </tr>
  <tr>
    <td>
      <p><code>validate.foundationsUnavailable</code></p>
    </td>
    <td>
<div class="highlight highlight-source-go"><pre>
type PrecheckerEventData struct {
	Environment config.Environment
	Description string
}
</pre></div>
    </td>
  </tr>
</tbody>
</table>
#### Example event handling sample
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
  // Create a temporary logger until we can create the Creator
  logLevel, _ := logging.LogLevel(defaultLevel)
  log := logger.DefaultLogger(os.Stdout, logLevel, "deployadactyl-consumer")

  c, err := creator.New(defaultLevel, defaultConfig)
  if err != nil {
    log.Fatal(err)
  }

  // This is an optional logger that makes logs look nice
  l := c.CreateLogger()

  // This is an optional event handling example
  em := c.CreateEventManager()

  myPackageHandler := mypackager.CreateMyPackageHandler()
  em.AddHandler(myPackageHandler, "deploy.start")
  em.AddHandler(myPackageHandler, "deploy.finish")
  em.AddHandler(myPackageHandler, "deploy.error")

  hf := c.CreateHandlerFunc()

  listener := c.CreateListener()
  l.Infof("Listening on Port %d", c.CreateConfig().Port)

  err = http.Serve(listener, hf)
  if err != nil {
    l.Fatal(err)
  }
}
```

## How to push your app

## How it works

Deployadactyl works by utilizing the [Cloud Foundry CLI](http://docs.cloudfoundry.org/cf-cli/) to push your application. It grabs a list of foundations from the Deployadactyl config, logs into each one and calls `cf push`. The general flow is to fetch your artifact, unzip it, and push it. Deployadactyl utilizes [blue green deployments](https://docs.pivotal.io/pivotalcf/devguide/deploy-apps/blue-green.html) and if it's unable to push your application it will rollback to the previous version.

## Contributing

- Fork the project
- Make a branch
- Commit to the branch
- Send us a Pull Request
