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

const mockRaids = [
  {
    id: "raid-1",
    name: "JavaScript",
    type: "Dynamic / JIT Compiled",
    players: "5/10",
    timer: "12:34",
    image: "/images/raid-javascript.png",
  },
  {
    id: "raid-2",
    name: "Rust",
    type: "Static / Compiled",
    players: "8/10",
    timer: "05:12",
    image: "/images/raid-rust.png",
  },
  {
    id: "raid-3",
    name: "Go",
    type: "Static / Compiled",
    players: "3/10",
    timer: "23:45",
    image: "/images/raid-go.png",
  },
];

export const handlers = [
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

  // JoinRaid (Unary)
  http.post(`${baseUrl}/raid_lobby.v1.RaidLobbyService/JoinRaid`, async ({ request }) => {
    const body = (await request.json()) as { lobbyId?: string; userId?: string };
    console.log("[MSW] JoinRaid:", body);

    return HttpResponse.json({
      participantId: `participant-${Math.random().toString(36).substring(7)}`,
    });
  }),

  // StartBattle (Unary)
  http.post(`${baseUrl}/raid_lobby.v1.RaidLobbyService/StartBattle`, async ({ request }) => {
    const body = (await request.json()) as { lobbyId?: string };
    console.log("[MSW] StartBattle:", body);

    return HttpResponse.json({
      battleSessionId: `battle-${Math.random().toString(36).substring(7)}`,
    });
  }),

  // ListRaids (Unary)
  http.post(`${baseUrl}/raid_lobby.v1.RaidLobbyService/ListOpenRaids`, async () => {
    return HttpResponse.json({
      raids: mockRaids,
    });
  }),

  // NOTE: StreamLobby (Server Streaming) は MSW では完全にモックするのは困難
  // 開発時は実際のバックエンドを起動するか、useLobbyStream を直接モック差し替えする
];
