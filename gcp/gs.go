package gcp

import (
	"context"

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
