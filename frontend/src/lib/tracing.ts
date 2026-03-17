import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-http";
import { registerInstrumentations } from "@opentelemetry/instrumentation";
import { FetchInstrumentation } from "@opentelemetry/instrumentation-fetch";
import { resourceFromAttributes } from "@opentelemetry/resources";
import { BatchSpanProcessor } from "@opentelemetry/sdk-trace-base";
import { WebTracerProvider } from "@opentelemetry/sdk-trace-web";
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from "@opentelemetry/semantic-conventions";
import { buildApiCorsPattern } from "./runtime-config";

export function initTracing() {
  const endpoint = import.meta.env.VITE_OTEL_ENDPOINT;
  if (!endpoint) {
    console.info("VITE_OTEL_ENDPOINT not set, tracing disabled");
    return;
  }

  const provider = new WebTracerProvider({
    resource: resourceFromAttributes({
      [ATTR_SERVICE_NAME]: "frontend",
      [ATTR_SERVICE_VERSION]: "1.0.0",
    }),
    spanProcessors: [
      new BatchSpanProcessor(new OTLPTraceExporter({ url: `${endpoint}/v1/traces` })),
    ],
  });

  provider.register();

  registerInstrumentations({
    instrumentations: [
      new FetchInstrumentation({
        propagateTraceHeaderCorsUrls: buildApiCorsPattern(),
        clearTimingResources: true,
      }),
    ],
  });

  console.info("OpenTelemetry tracing initialized");
}
