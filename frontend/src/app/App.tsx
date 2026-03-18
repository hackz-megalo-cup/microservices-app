import { BrowserRouter, Route, Routes } from "react-router";
import { ApiTestPage } from "../features/api-test/components/api-test-page";
import { LoginPage } from "../features/auth/components/login-page";
import { RequireAuth } from "../features/auth/components/require-auth";
import { BattlePage } from "../features/battle/components/battle-page";
import { RaidTestPage } from "../features/raid-test/components/raid-test-page";
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
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <Home />
            </RequireAuth>
          }
        />
        <Route
          path="/battle/:id"
          element={
            <RequireAuth>
              <BattlePage />
            </RequireAuth>
          }
        />
        <Route
          path="/lobby/:id"
          element={
            <RequireAuth>
              <Lobby />
            </RequireAuth>
          }
        />
        <Route
          path="/victory/:id"
          element={
            <RequireAuth>
              <Victory />
            </RequireAuth>
          }
        />
        <Route
          path="/capture/:id"
          element={
            <RequireAuth>
              <Capture />
            </RequireAuth>
          }
        />
        <Route
          path="/collection"
          element={
            <RequireAuth>
              <Collection />
            </RequireAuth>
          }
        />
        <Route
          path="/collection/:id"
          element={
            <RequireAuth>
              <Detail />
            </RequireAuth>
          }
        />
        <Route path="/api-test" element={<ApiTestPage />} />
        <Route path="/raid-test" element={<RaidTestPage />} />
      </Routes>
    </BrowserRouter>
  );
}
