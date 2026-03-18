import { useQuery } from "@tanstack/react-query";
import type { Raid } from "../types";
import "../../../styles/global.css";
import { RaidCard } from "./ui/raid-card";
import { TabBar } from "./ui/tab-bar";

// TODO: proto実装後、以下に置き換え
// import { useQuery } from "@connectrpc/connect-query";
// import { listRaids } from "../../../gen/raid_lobby/v1/raid_lobby-RaidLobbyService_connectquery";

async function fetchListRaids(): Promise<{ raids: Raid[] }> {
  const response = await fetch(`${apiBaseUrl}/raid_lobby.v1.RaidLobbyService/ListRaids`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({}),
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch raids: ${response.statusText}`);
  }

  return response.json() as Promise<{ raids: Raid[] }>;
}

export function Home() {
  // TODO: proto実装後、以下に置き換え
  // const { data, isLoading, error } = useQuery(listRaids, {});
  const { data, isLoading, error } = useQuery({
    queryKey: ["listRaids"],
    queryFn: fetchListRaids,
  });

  const raids = data?.raids ?? [];

  return (
    <div className="showcase-screen">
      <header className="flex items-center justify-between px-6 py-3">
        <span className="text-xs font-bold tracking-widest text-text-secondary">POKÉMON</span>
        <button
          type="button"
          className="flex items-center justify-center w-11 h-11 bg-transparent text-xl cursor-pointer rounded-lg hover:bg-bg-hover"
          aria-label="Notifications"
        >
          🔔
        </button>
      </header>

      <section
        className="flex flex-col items-center justify-center gap-3 h-[280px]"
        style={{ background: "radial-gradient(circle, var(--color-accent-glow), transparent)" }}
      >
        <img
          src="/images/hero-python.png"
          alt="Python"
          className="w-[200px] h-[200px] rounded-full object-cover"
        />
        <h1 className="text-2xl font-bold text-text-primary m-0">Python</h1>
        <p className="text-sm text-text-secondary m-0">Dynamic / Interpreted</p>
      </section>

      <section className="flex flex-col gap-3 px-6">
        <span className="text-xs font-bold tracking-widest text-text-secondary">ACTIVE RAIDS</span>
        {isLoading && <p className="text-sm text-text-secondary">Loading raids...</p>}
        {error && <p className="text-sm text-red-500">Error: {error.message}</p>}
        {raids.map((raid) => (
          <RaidCard
            key={raid.id}
            id={raid.id}
            name={raid.name}
            type={raid.type}
            players={raid.players}
            timer={raid.timer}
            image={raid.image}
          />
        ))}
      </section>

      <div className="flex-1" />

      <TabBar active="HOME" />
    </div>
  );
}
