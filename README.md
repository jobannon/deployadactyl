Deployadactyl deploys applications to multiple Cloud Foundry instances. If a deployment fails in any instance, it will automatically rollback.

## How to push your app

## How it works

Deployadactyl works by utilizing the [Cloud Foundry CLI](http://docs.cloudfoundry.org/cf-cli/) to push your application. It grabs a list of foundations from the Deployadactyl config, logs into each one and calls `cf push`. The general flow is to fetch your artifact, unzip it, and push it. Deployadactyl utilizes [blue green deployments](https://docs.pivotal.io/pivotalcf/devguide/deploy-apps/blue-green.html) and if it's unable to push your application it will rollback to the previous version.

## Contributing

- Fork the project
- Make a branch
- Commit to the branch
- Send us a Pull Request
