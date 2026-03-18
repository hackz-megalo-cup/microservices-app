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

const mockOpenRaids = [
  {
    id: "raid-1",
    bossPokemonId: "1",
    currentParticipants: 5,
    maxParticipants: 10,
    status: "waiting",
    createdAt: "2026-03-18T10:48:00Z",
  },
  {
    id: "raid-2",
    bossPokemonId: "2",
    currentParticipants: 8,
    maxParticipants: 10,
    status: "waiting",
    createdAt: "2026-03-18T10:55:00Z",
  },
  {
    id: "raid-3",
    bossPokemonId: "3",
    currentParticipants: 3,
    maxParticipants: 10,
    status: "in_battle",
    createdAt: "2026-03-18T10:37:00Z",
  },
];

const mockItems = [
  {
    id: "capture-ball-basic",
    name: "Basic Ball",
    effects: [
      {
        effectType: "capture_rate_up",
        targetType: "",
        captureRateBonus: 0.1,
        flavorText: "Basic Poké Ball",
      },
    ],
  },
  {
    id: "capture-ball-super",
    name: "Super Ball",
    effects: [
      {
        effectType: "capture_rate_up",
        targetType: "",
        captureRateBonus: 0.2,
        flavorText: "Super Poké Ball",
      },
    ],
  },
  {
    id: "capture-ball-ultra",
    name: "Ultra Ball",
    effects: [
      {
        effectType: "capture_rate_up",
        targetType: "",
        captureRateBonus: 0.35,
        flavorText: "Ultra Poké Ball",
      },
    ],
  },
];

const mockInventoryByUser: Record<
  string,
  Array<{ itemId: string; quantity: number; status: string }>
> = {
  demo: [
    { itemId: "capture-ball-basic", quantity: 5, status: "active" },
    { itemId: "capture-ball-super", quantity: 2, status: "active" },
  ],
};

function pickString(body: unknown, keys: string[]): string {
  if (!body || typeof body !== "object") {
    return "";
  }
  const record = body as Record<string, unknown>;
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string") {
      return value;
    }
  }
  return "";
}

function pickNumber(body: unknown, keys: string[]): number | null {
  if (!body || typeof body !== "object") {
    return null;
  }
  const record = body as Record<string, unknown>;
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "number") {
      return value;
    }
  }
  return null;
}

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

  http.post(`${baseUrl}/masterdata.v1.MasterdataService/ListItems`, () => {
    return HttpResponse.json({
      items: mockItems,
    });
  }),

  // JoinRaid (Unary)
  http.post(`${baseUrl}/raid_lobby.v1.RaidLobbyService/JoinRaid`, async ({ request }) => {
    await request.json();

    return HttpResponse.json({
      participantId: `participant-${Math.random().toString(36).substring(7)}`,
    });
  }),

  // StartBattle (Unary)
  http.post(`${baseUrl}/raid_lobby.v1.RaidLobbyService/StartBattle`, async ({ request }) => {
    await request.json();

    return HttpResponse.json({
      battleSessionId: `battle-${Math.random().toString(36).substring(7)}`,
    });
  }),

  // ListRaids (Unary)
  http.post(`${baseUrl}/raid_lobby.v1.RaidLobbyService/ListOpenRaids`, async () => {
    return HttpResponse.json({
      raids: mockOpenRaids,
    });
  }),

  // NOTE: StreamLobby (Server Streaming) は MSW では完全にモックするのは困難
  // 開発時は実際のバックエンドを起動するか、useLobbyStream を直接モック差し替えする

  http.post(`${baseUrl}/item.v1.ItemService/GetUserItems`, async ({ request }) => {
    const body = await request.json();
    const userId = pickString(body, ["userId", "user_id"]);

    if (!userId) {
      return HttpResponse.json(
        {
          code: "invalid_argument",
          message: "user_id is required",
        },
        { status: 400 },
      );
    }

    return HttpResponse.json({
      items: mockInventoryByUser[userId] ?? [],
    });
  }),

  // CaptureService mock handlers
  http.post(`${baseUrl}/capture.v1.CaptureService/GetCaptureSession`, async ({ request }) => {
    const body = await request.json();
    const sessionId = pickString(body, ["sessionId", "session_id"]);

    return HttpResponse.json({
      sessionId: sessionId || "mock-session-1",
      battleSessionId: "mock-battle-1",
      userId: "demo",
      pokemonId: "1",
      baseRate: 0.3,
      currentRate: 0.3,
      result: "pending",
      actions: [],
    });
  }),

  http.post(`${baseUrl}/capture.v1.CaptureService/UseItem`, async ({ request }) => {
    const body = await request.json();
    const itemId = pickString(body, ["itemId", "item_id"]);

    // Simulate escape effect for specific item (for testing)
    if (itemId === "escape-item") {
      return HttpResponse.json({
        rateBefore: 0.3,
        rateAfter: 0.3,
        escaped: true,
        flavorText: "The Pokémon fled!",
      });
    }

    return HttpResponse.json({
      rateBefore: 0.3,
      rateAfter: 0.5,
      escaped: false,
      flavorText: "Capture rate increased!",
    });
  }),

  http.post(`${baseUrl}/capture.v1.CaptureService/ThrowBall`, async () => {
    const success = Math.random() < 0.5;
    return HttpResponse.json({
      result: success ? "success" : "fail",
    });
  }),

  http.post(`${baseUrl}/capture.v1.CaptureService/EndSession`, async ({ request }) => {
    const body = await request.json();
    const sessionId = pickString(body, ["sessionId", "session_id"]);
    return HttpResponse.json({
      result: sessionId ? "completed" : "completed",
    });
  }),

  http.post(`${baseUrl}/item.v1.ItemService/UseItem`, async ({ request }) => {
    const body = await request.json();
    const userId = pickString(body, ["userId", "user_id"]);
    const itemId = pickString(body, ["itemId", "item_id"]);
    const quantity = pickNumber(body, ["quantity"]) ?? 0;

    if (!userId || !itemId || quantity <= 0) {
      return HttpResponse.json(
        {
          code: "invalid_argument",
          message: "user_id, item_id and positive quantity are required",
        },
        { status: 400 },
      );
    }

    if (itemId === "force-unimplemented") {
      return HttpResponse.json(
        {
          code: "unimplemented",
          message: "UseItem is not implemented",
        },
        { status: 501 },
      );
    }

    const inventory = mockInventoryByUser[userId] ?? [];
    const index = inventory.findIndex((entry) => entry.itemId === itemId);

    if (index < 0) {
      return HttpResponse.json(
        {
          code: "not_found",
          message: "item not found for user",
        },
        { status: 404 },
      );
    }

    if (inventory[index].quantity < quantity) {
      return HttpResponse.json(
        {
          code: "failed_precondition",
          message: "not enough quantity",
        },
        { status: 412 },
      );
    }

    inventory[index].quantity -= quantity;
    mockInventoryByUser[userId] = inventory.filter((entry) => entry.quantity > 0);

    return HttpResponse.json({});
  }),
];
