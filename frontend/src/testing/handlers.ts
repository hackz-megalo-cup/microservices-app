import { HttpResponse, http } from "msw";

const baseUrl =
  import.meta.env.VITE_API_BASE_URL ||
  (typeof window !== "undefined" ? window.location.origin : "http://localhost:8080");

export const handlers = [
  http.post(`${baseUrl}/greeter.v1.GreeterService/Greet`, async ({ request }) => {
    const body = (await request.json()) as { name?: string };
    const name = body.name || "World";

    return HttpResponse.json({
      message: `Hello ${name} from mocked greeter!`,
      externalStatus: 200,
      externalBodyLength: 321,
    });
  }),
  http.post(`${baseUrl}/gateway.v1.GatewayService/InvokeCustom`, async ({ request }) => {
    const body = (await request.json()) as { name?: string };
    const name = body.name || "World";

    return HttpResponse.json({
      message: `Hello ${name} from mocked gateway!`,
    });
  }),
];
