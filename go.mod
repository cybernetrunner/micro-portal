module micro/portal

go 1.16

replace github.com/spf13/afero => github.com/spf13/afero v1.5.1

require (
	github.com/envoyproxy/protoc-gen-validate v0.6.2
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.2
	github.com/infobloxopen/atlas-app-toolkit v1.1.2
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	google.golang.org/genproto v0.0.0-20211222154725-9823f7ba7562
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.27.1
)
