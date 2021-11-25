## Deploying a Devnet

This document explains the flow of deploying a devnet and web services.

### Authentication

First you need to set up authentication to GCP. 


```
export SPACECRAFT_GCP_PROJECT=spacecraft-id
export SPACECRAFT_GCP_LOCATION=us-central1
export SPACECRAFT_GCP_ZONE=us-central1-b
export GOOGLE_APPLICATION_CREDENTIALS=~/spacecraft-id-f8aeb7a6d337.json
```

And then run the below command to activate gcloud to use the service account:

```
gcloud auth activate-service-account --key-file ~/spacecraft-id-f8aeb7a6d337.json
```

### Set Cloudflare API Token

Set cloudflare API token in ENV so that all the services can be exposed publicly: 

```
export SPACECRAFT_CLOUDFLARE_API_TOKEN=
```

You can find the token in 1password

### Creating Network

Clone go-spacecraft and build the CLI using the below commands:

```
git clone git@github.com:spacemeshos/go-spacecraft.git
cd go-spacecraft
go build
```

Then you can edit the network id in the ./artifacts/devnet/miner/config.json file and finally inside the go-spacecraft run the below command to deploy a devnet:

```
./go-spacecraft createNetwork --config=./artifacts/devnet/config.json --network-name=devnet
```

If you want to change go-spacemesh config then you can change that in ./artifacts/devnet/miner/config.json and if you want to change infrastructure config then you can change that in ./artifacts/devnet/config.json. You can also run “./gospacecraft --help” to know all the available CLI options.

The process takes 3-5 minutes to finish and then at the end Kibana and Pyroscope URLs will be printed. 

You can find the newly deployed k8s cluster here https://console.cloud.google.com/kubernetes/list?authuser=1&project=spacecraft-id

### Configure Kubectl

To point kubectl to the deployed k8s cluster run the below command:

```
gcloud container clusters get-credentials devnet --region us-central1 --project spacecraft-id
```

### Metrics

Promethues stack is deployed by default and metrics is enabled in managed miners. Here is the credentials for grafana:

Username: admin
Password: prom-operator

### Smapp Config

When you deploy a network using go-spacecraft it creates and uploads a config file to GCP storage. You can access it using https://console.cloud.google.com/storage/browser/spacecraft-data/devnet-archive

### Slack Alerts

To enable slack alerts you need to set the following environment variables:

```
export SPACECRAFT_SLACK_CHANNEL_ID=
export SPACECRAFT_SLACK_TOKEN=
```

Channel ID is the channel where you want to publish the alerts to.


### Deploy Web Services

To deploy web services first you first need to place the secret `tls.key` and `tls.crt` files in ./artifacts/ws directory. These are the TLS secret and signed certificate files purchased from TLS certificate provider for spacemesh.io domain. These files are needed to secure GRPC connection of public go-spacemesh API with TLS. 


Then you can run the below command:

```
./go-spacecraft deployWS --config=--config=./artifacts/devnet/config.json --network-name=devnet
```


