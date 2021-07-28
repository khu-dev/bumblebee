package main

var (
	Config *BumblebeeConfig = &BumblebeeConfig{}
)

type BumblebeeConfig struct {
	Host                    string
	Port                    int
	NumOfTransformerWorkers int
	GracefulShutdown struct{
		MaxTimeout int
		UploaderTimeout int
	}
	Storage                 struct {
		Aws struct {
			Enabled    bool
			BucketName string
			Endpoint   string
		}
		Disk struct {
			Enabled  bool
			RootPath string
		}
	}
}
