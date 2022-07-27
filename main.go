package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	redirectTo   = os.Getenv("REDIRECT_TO")
	listenAndServe   = os.Getenv("LISTEN_AND_SERVE")
	serviceName  = os.Getenv("SERVICE_NAME")
	collectorURL = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	insecure     = os.Getenv("INSECURE_MODE")
)

var tracer = otel.Tracer(serviceName)

func proxy(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s - %s", r.Method, r.URL.Path), oteltrace.WithAttributes())
	defer span.End()

	client := &http.Client{}

	req, err := http.NewRequest(r.Method, redirectTo, r.Body)
	if err != nil {
		log.Fatalln(err)
	}

	mapReq(r, req)

	_, spanReq := tracer.Start(ctx, "HttpReq", traceReqOptions(r))
	spanReq.End()

	_, spanProcessing := tracer.Start(ctx, "Processing", oteltrace.WithAttributes())
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	spanProcessing.End()

	mapResp(resp, w)

	_, spanResp := tracer.Start(ctx, "HttpResp", traceRespOptions(resp))
	spanResp.End()
}

func main() {
	cleanup := initTracer()
	defer cleanup(context.Background())

	http.HandleFunc("/", proxy)

	http.ListenAndServe(listenAndServe, nil)
}

func mapReq(from *http.Request, to *http.Request) {
	to.URL.Path = from.URL.Path
	to.Header = from.Header
	to.Body = from.Body
}

func mapResp(from *http.Response, to http.ResponseWriter) {
	bodyResp, err := ioutil.ReadAll(from.Body)
	if err != nil {
		log.Fatalln(err)
	}
	from.Body = ioutil.NopCloser(bytes.NewReader(bodyResp))

	to.WriteHeader(from.StatusCode)
	to.Write(bodyResp)

	for header, values := range from.Header {
		for i, value := range values {
			if i == 0 {
				to.Header().Set(header, value)
			} else {
				to.Header().Add(header, value)
			}
		}
	}
}

func traceReqOptions(r *http.Request) oteltrace.SpanStartEventOption {
	bodyReq, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(bodyReq))

	strHeadersReq, err := json.Marshal(r.Header)
	if err != nil {
		log.Fatalln(err)
	}

	return oteltrace.WithAttributes(
		attribute.String("Path", r.URL.Path),
		attribute.String("Headers", string(strHeadersReq)),
		attribute.String("Body", string(bodyReq)))
}

func traceRespOptions(r *http.Response) oteltrace.SpanStartEventOption {
	bodyReq, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}
	strHeadersReq, err := json.Marshal(r.Header)
	if err != nil {
		log.Fatalln(err)
	}

	return oteltrace.WithAttributes(
		attribute.Int("Status", r.StatusCode),
		attribute.String("Headers", string(strHeadersReq)),
		attribute.String("Body", string(bodyReq)))
}

func initTracer() func(context.Context) error {

	secureOption := otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	if len(insecure) > 0 {
		secureOption = otlptracegrpc.WithInsecure()
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(collectorURL),
		),
	)

	if err != nil {
		log.Fatal(err)
	}
	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		log.Printf("Could not set resources: ", err)
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)
	return exporter.Shutdown
}
