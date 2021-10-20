package gcp

import (
	"context"
	"io/ioutil"
	"time"

	"cloud.google.com/go/storage"
)

func UploadConfig(fileContent string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)

	if err != nil {
		return err
	}

	defer client.Close()

	wc := client.Bucket("spacecraft-data").Object(config.NetworkName + "-archive" + "/config.json").NewWriter(ctx)

	if _, err := wc.Write([]byte(fileContent)); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

func ReadConfig() (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)

	if err != nil {
		return "", err
	}

	defer client.Close()

	rc, err := client.Bucket("spacecraft-data").Object(config.NetworkName + "-archive" + "/config.json").NewReader(ctx)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	body, err := ioutil.ReadAll(rc)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func ReadWSConfig() (string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)

	if err != nil {
		return "", err
	}

	defer client.Close()

	rc, err := client.Bucket("sm-discovery-service").Object("index.json").NewReader(ctx)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	body, err := ioutil.ReadAll(rc)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func UploadWSConfig(fileContent string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)

	if err != nil {
		return err
	}

	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	o := client.Bucket("sm-discovery-service").Object("index.json")
	if err := o.Delete(ctx); err != nil {
		return err
	}

	wc := client.Bucket("sm-discovery-service").Object("index.json").NewWriter(ctx)

	if _, err := wc.Write([]byte(fileContent)); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}
