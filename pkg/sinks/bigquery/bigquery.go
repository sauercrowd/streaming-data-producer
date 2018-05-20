package bigquery

import (
	"context"
	"fmt"

	"github.com/sauercrowd/streaming-data-producer/pkg/data"
	"github.com/sauercrowd/streaming-data-producer/pkg/sources/spotify"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
)

type Sink struct {
	uploader *bigquery.Uploader
}

func (bq *Sink) init(ctx context.Context, projectID, datasetName, tableName string, schemaStruct interface{}) error {
	// Creates a client.
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Failed to create client: %v", err)
	}

	// Creates the new BigQuery dataset.
	if err := client.Dataset(datasetName).Create(ctx, &bigquery.DatasetMetadata{}); err != nil && err.(*googleapi.Error).Code != 409 {
		return fmt.Errorf("Failed to create dataset: %v", err)
	}

	fmt.Printf("Dataset created\n")

	dataset := client.DatasetInProject(projectID, datasetName)

	//structType := reflect.TypeOf(schemaStruct)
	//specialisedStruct := reflect.ValueOf(schemaStruct).Convert(structType)
	schema, err := bigquery.InferSchema(spotify.CurrentlyPlayingStruct{})
	if err != nil {
		return fmt.Errorf("Failed to infer schema: %v", err)
	}

	table := dataset.Table(tableName)
	if err := table.Create(ctx, &bigquery.TableMetadata{Schema: schema}); err != nil && err.(*googleapi.Error).Code != 409 {
		return fmt.Errorf("Faild to create table: %v", err)
	}

	bq.uploader = table.Uploader()
	return nil
}

func (bq *Sink) Listen(ctx context.Context, projectID, datasetName, tableName string, ch chan data.Datapoint) error {
	first := true
	for datapoint := range ch {
		if first {
			if err := bq.init(ctx, projectID, datasetName, tableName, datapoint.Struct); err != nil {
				return err
			}
		}
		// structType := reflect.TypeOf(datapoint.Struct)
		// specialisedStruct := reflect.ValueOf(datapoint.Struct).Convert(structType)
		x, _ := datapoint.Struct.(spotify.CurrentlyPlayingStruct)
		items := []*spotify.CurrentlyPlayingStruct{&x}
		if err := bq.uploader.Put(ctx, items); err != nil {
			return err
		}
	}
	return nil
}
