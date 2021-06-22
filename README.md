# Spacecraft

A CLI tool to deploy and manage Spacemesh Network on GKE. 

The default configuration of the tool lets you launch a mininet i.e., a network consisting of 10 miners and 1 poet. Mininet is helpful for development and testing of spacecraft tool itself but to test go-spacemesh or poet builds you need to launch a devnet as it has configured to enable to network run for long time.

## Launching a Devnet

To launch a devnet follow the below steps:

### Install Go

This tool requires go 1.15 or above installed. If you haven't installed golang then follow the instructions here https://golang.org/doc/install

### Set secrets in ENV

Set the following environment variables:

```
SPACECRAFT_GCP_PROJECT=...
SPACECRAFT_GCP_LOCATION=...
SPACECRAFT_GCP_ZONE=...
GOOGLE_APPLICATION_CREDENTIALS=...
```

To learn more on how to set the `GOOGLE_APPLICATION_CREDENTIALS` variable refer this link https://cloud.google.com/docs/authentication/getting-started

`SPACECRAFT_GCP_PROJECT` is the project id which you want to use for this tool and `SPACECRAFT_GCP_LOCATION`/`SPACECRAFT_GCP_ZONE` indicates the location and zone respectively for the GKE cluster.

### Clone and Build CLI

Clone the repository and build the CLI binary. Run the below commands to do that:

```
git clone git@github.com:spacemeshos/go-spacecraft.git
cd go-spacecraft
go build
```

### Create the network

The final step is to create the network. Run the below command to create a devnet:

```
./go-spacecraft createNetwork --config=./artifacts/devnet/config.json
```

This command will take around ~10min to finish and at the end you should see the kibana URL. The generated config file is uploaded to GCP storage. 

Note that its not recommended to move the `go-spacecraft` binary outside the repository as it looks for files in artifcats directory. In case you want to move it then make sure you provide the artifacts directory path in the required CLI flags.
