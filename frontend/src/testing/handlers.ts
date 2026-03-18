import { HttpResponse, http } from "msw";

const baseUrl =
  import.meta.env.VITE_API_BASE_URL ||
  (typeof window !== "undefined" ? window.location.origin : "http://localhost:8080");

const mockPokemon = [
  {
    id: "1",
    name: "Pikachu",
    type: "Electric",
    hp: 35,
    attack: 55,
    speed: 90,
    specialMoveName: "Thunderbolt",
    specialMoveDamage: 90,
  },
  {
    id: "2",
    name: "Charizard",
    type: "Fire",
    hp: 78,
    attack: 84,
    speed: 100,
    specialMoveName: "Flamethrower",
    specialMoveDamage: 90,
  },
  {
    id: "3",
    name: "Blastoise",
    type: "Water",
    hp: 79,
    attack: 83,
    speed: 78,
    specialMoveName: "Hydro Pump",
    specialMoveDamage: 110,
  },
  {
    id: "4",
    name: "Venusaur",
    type: "Grass",
    hp: 80,
    attack: 82,
    speed: 80,
    specialMoveName: "Solar Beam",
    specialMoveDamage: 120,
  },
  {
    id: "5",
    name: "Gengar",
    type: "Ghost",
    hp: 60,
    attack: 65,
    speed: 110,
    specialMoveName: "Shadow Ball",
    specialMoveDamage: 80,
  },
];

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
  http.post(`${baseUrl}/masterdata.v1.MasterdataService/ListPokemon`, () => {
    return HttpResponse.json({
      pokemon: mockPokemon,
    });
  }),
  http.post(`${baseUrl}/masterdata.v1.MasterdataService/GetPokemon`, async ({ request }) => {
    const body = (await request.json()) as { id?: string };
    const target = mockPokemon.find((pokemon) => pokemon.id === body.id);

    if (!target) {
      return HttpResponse.json(
        {
          code: "not_found",
          message: "pokemon not found",
        },
        { status: 404 },
      );
    }

    return HttpResponse.json({
      pokemon: target,
    });
  }),
];
