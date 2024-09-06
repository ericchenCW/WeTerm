package action

import (
	"context"
	"strings"
	"weterm/model"

	"github.com/rivo/tview"
	logzero "github.com/rs/zerolog/log"

	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func SendTraceView(receiver *model.AppModel) {
	output := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		ScrollToEnd()
	output.SetBorder(true).SetTitle("Output").SetTitleAlign(tview.AlignLeft)
	form := tview.NewForm()
	form.AddInputField("Endpoint", "127.0.0.1:4317", 20, nil, nil)
	form.AddInputField("Attributes", "bk.biz.id=2,probe.name=0000-0000-0000-0000", 0, nil, nil)
	form.AddButton("Send Trace", func() {
		endpoint := form.GetFormItemByLabel("Endpoint").(*tview.InputField).GetText()
		attributes := form.GetFormItemByLabel("Attributes").(*tview.InputField).GetText()
		Send(endpoint, attributes, output)
	})
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(output, 0, 2, true)
	layout.SetBorder(true).SetTitle("发送trace").SetTitleAlign(tview.AlignCenter)
	receiver.CorePages.AddPage("trace_page", layout, true, false)
	receiver.CorePages.SwitchToPage("trace_page")
}

func Send(endpoint string, attributes string, output *tview.TextView) {
	// 设置自定义日志记录器
	log.SetOutput(&TextViewLogger{view: output})
	// Initialize OpenTelemetry Tracer
	tracerProvider, err := initTracer(endpoint)
	if err != nil {
		logzero.Error().Msg("failed to initialize tracer provider")
		return
	}
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			log.Default().Println("failed to shutdown trace provider")
		} else {
			log.Default().Println("Send done.")
		}
	}()

	tracer := otel.Tracer("test-cli-tracer")

	// Parse attributes
	var attrs []attribute.KeyValue
	if attributes != "" {
		pairs := strings.Split(attributes, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				logzero.Error().Msg("invalid attribute format")
				return
			}
			attrs = append(attrs, attribute.String(parts[0], parts[1]))
		}
	}

	// Create a span
	_, span := tracer.Start(context.Background(), "test-cli",
		trace.WithAttributes(attrs...),
	)
	span.AddEvent("helloworld")
	span.End()
	// log info endponit and attributes in one line
	log.Default().Println("Sending trace to", "endpoint:", endpoint, "attributes:", attributes, "done.")
}

type TextViewLogger struct {
	view *tview.TextView
}

func (l *TextViewLogger) Write(p []byte) (n int, err error) {
	l.view.Write(p)
	return len(p), nil
}

func initTracer(endpoint string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// Create OTLP gRPC exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return nil, err
	}

	// Create trace provider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			attribute.String("service.name", "test-cli-service"),
		)),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

type CustomSpanProcessor struct {
	output *tview.TextView
}
