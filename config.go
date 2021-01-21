package main

var (
	Config *BumblebeeConfig
)

type BumblebeeConfig struct {
	S3BucketName            string
	NumOfTransformerWorkers int
}

func InitConfig() {
	Config = &BumblebeeConfig{
		NumOfTransformerWorkers: 2,
	}
}
