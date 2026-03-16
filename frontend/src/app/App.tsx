import { BrowserRouter, Route, Routes } from "react-router";
import { ApiTestPage } from "../features/api-test/components/ApiTestPage";
import { Battle } from "../features/showcase/components/Battle";
import { Capture } from "../features/showcase/components/Capture";
import { Collection } from "../features/showcase/components/Collection";
import { Detail } from "../features/showcase/components/Detail";
import { Home } from "../features/showcase/components/Home";
import { Lobby } from "../features/showcase/components/Lobby";
import { Victory } from "../features/showcase/components/Victory";

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
      </Routes>
    </BrowserRouter>
  );
}
