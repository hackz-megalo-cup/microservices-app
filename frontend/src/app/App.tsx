import { BrowserRouter, Route, Routes } from "react-router";
import { ApiTestPage } from "../features/api-test/components/api-test-page";
import { RaidTestPage } from "../features/raid-test/components/raid-test-page";
import { Battle } from "../features/showcase/components/battle";
import { Capture } from "../features/showcase/components/capture";
import { Collection } from "../features/showcase/components/collection";
import { Detail } from "../features/showcase/components/detail";
import { Home } from "../features/showcase/components/home";
import { Lobby } from "../features/showcase/components/lobby";
import { Victory } from "../features/showcase/components/victory";

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/battle/:id" element={<Battle />} />
        <Route path="/lobby/:id" element={<Lobby />} />
        <Route path="/victory/:id" element={<Victory />} />
        <Route path="/capture/:id" element={<Capture />} />
        <Route path="/collection" element={<Collection />} />
        <Route path="/collection/:id" element={<Detail />} />
        <Route path="/api-test" element={<ApiTestPage />} />
        <Route path="/raid-test" element={<RaidTestPage />} />
      </Routes>
    </BrowserRouter>
  );
}
