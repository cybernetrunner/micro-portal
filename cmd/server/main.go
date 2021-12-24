package main

import (
	"context"
	"fmt"
	"log"
	"micro/portal/pkg/pb"
	"net"
	"net/http"
	"strings"

	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/infobloxopen/atlas-app-toolkit/gateway"
	"github.com/infobloxopen/atlas-app-toolkit/server"

	"github.com/infobloxopen/atlas-app-toolkit/gorm/resource"
)

var (
	logLevel = viper.GetString("logging.level")

	// internal configuration variables
	internalEnabled   = viper.GetBool("internal.enable")
	internalAddress   = viper.GetString("internal.address")
	internalPort      = viper.GetString("internal.port")
	internalServeHost = fmt.Sprintf("%s:%s", internalAddress, internalPort)

	// gateway configuration variables
	gatewayEndpoint    = viper.GetString("gateway.endpoint")
	gatewaySwaggerFile = viper.GetString("gateway.swaggerFile")
	gatewayAddress     = viper.GetString("gateway.address")
	gatewayPort        = viper.GetString("gateway.port")
	gatewayServeHost   = fmt.Sprintf("%s:%s", gatewayAddress, gatewayPort)

	// server configuration variables
	serverAddress   = viper.GetString("server.address")
	serverPort      = viper.GetString("server.port")
	serverServeHost = fmt.Sprintf("%s:%s", serverAddress, serverPort)
)

func main() {
	doneC := make(chan error)
	logger := NewLogger()
	if internalEnabled {
		go func() { doneC <- ServeInternal(logger) }()
	}

	go func() { doneC <- ServeExternal(logger) }()

	if err := <-doneC; err != nil {
		logger.Fatal(err)
	}
}

func NewLogger() *logrus.Logger {
	logger := logrus.StandardLogger()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logger.SetReportCaller(true)

	// Set the log level on the default logger based on command line flag
	if level, err := logrus.ParseLevel(logLevel); err != nil {
		logger.Errorf("Invalid %q provided for log level", logLevel)
		logger.SetLevel(logrus.InfoLevel)
	} else {
		logger.SetLevel(level)
	}

	return logger
}

// ServeInternal builds and runs the server that listens on InternalAddress
func ServeInternal(logger *logrus.Logger) error {

	s, err := server.NewServer(
		// register metrics
		server.WithHandler("/metrics", promhttp.Handler()),
	)
	if err != nil {
		return err
	}
	l, err := net.Listen("tcp", internalServeHost)
	if err != nil {
		return err
	}

	logger.Debugf("serving internal http at %q", internalServeHost)
	return s.Serve(nil, l)
}

// ServeExternal builds and runs the server that listens on ServerAddress and GatewayAddress
func ServeExternal(logger *logrus.Logger) error {

	grpcServer, err := NewGRPCServer(logger)
	if err != nil {
		logger.Fatalln(err)
	}
	grpcPrometheus.Register(grpcServer)

	s, err := server.NewServer(
		server.WithGrpcServer(grpcServer),
		server.WithGateway(
			gateway.WithGatewayOptions(
				runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
					MarshalOptions: protojson.MarshalOptions{
						UseProtoNames:   true,
						EmitUnpopulated: false,
					},
				}),
				runtime.WithForwardResponseOption(forwardResponseOption),
				runtime.WithIncomingHeaderMatcher(gateway.AtlasDefaultHeaderMatcher()),
			),
			gateway.WithServerAddress(internalServeHost),
			gateway.WithEndpointRegistration(gatewayEndpoint, pb.RegisterPortalHandlerFromEndpoint),
		),
		server.WithHandler("/swagger/", NewSwaggerHandler(gatewaySwaggerFile)),
	)
	if err != nil {
		logger.Fatalln(err)
	}

	grpcL, err := net.Listen("tcp", serverServeHost)
	if err != nil {
		logger.Fatalln(err)
	}

	httpL, err := net.Listen("tcp", gatewayServeHost)
	if err != nil {
		logger.Fatalln(err)
	}

	logger.Printf("serving gRPC at %s:%s", serverAddress, serverPort)
	logger.Printf("serving http at %s:%s", gatewayAddress, gatewayPort)

	return s.Serve(grpcL, httpL)
}

func init() {
	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.Fatalf("cannot load configuration: %v", err)
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AddConfigPath(viper.GetString("config.source"))

	if viper.GetString("config.file") != "" {
		log.Printf(
			"Serving from configuration file: %s",
			viper.GetString("config.file"),
		)
		viper.SetConfigName(viper.GetString("config.file"))
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("cannot load configuration: %v", err)
		}
	} else {
		log.Printf("Serving from default values, environment variables, and/or flags")
	}
	resource.RegisterApplication(viper.GetString("app.id"))
	resource.SetPlural()
}

func forwardResponseOption(ctx context.Context, w http.ResponseWriter, resp protoreflect.ProtoMessage) error {
	w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	return nil
}
