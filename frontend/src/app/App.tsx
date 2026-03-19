import { lazy, Suspense } from "react";
import { BrowserRouter, Route, Routes } from "react-router";
import {
  AdminDashboard,
  AdminLayout,
  ItemForm,
  ItemList,
  PokemonForm,
  PokemonList,
  RaidForm,
  RaidList,
  TypeMatchupList,
} from "../features/admin";
import { ApiTestPage } from "../features/api-test/components/api-test-page";
import { LoginPage } from "../features/auth/components/login-page";
import { RequireAuth } from "../features/auth/components/require-auth";
import { StarterSelect } from "../features/auth/components/starter-select";
import { RaidTestPage } from "../features/raid-test/components/raid-test-page";

const BattlePage = lazy(() =>
  import("../features/battle/components/battle-page").then((m) => ({ default: m.BattlePage })),
);

import { Capture } from "../features/showcase/components/capture";
import { CaptureDemo } from "../features/showcase/components/capture-demo";
import { Collection } from "../features/showcase/components/collection";
import { Detail } from "../features/showcase/components/detail";
import { Home } from "../features/showcase/components/home";
import { Lobby } from "../features/showcase/components/lobby";
import { Profile } from "../features/showcase/components/profile";
import { Victory } from "../features/showcase/components/victory";

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/starter-select"
          element={
            <RequireAuth>
              <StarterSelect />
            </RequireAuth>
          }
        />
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
              <Suspense
                fallback={
                  <div className="flex items-center justify-center h-screen text-text-secondary">
                    Loading...
                  </div>
                }
              >
                <BattlePage />
              </Suspense>
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
        <Route path="/capture/demo" element={<CaptureDemo />} />
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
        <Route
          path="/profile"
          element={
            <RequireAuth>
              <Profile />
            </RequireAuth>
          }
        />
        <Route
          path="/admin"
          element={
            <RequireAuth>
              <AdminLayout />
            </RequireAuth>
          }
        >
          <Route index element={<AdminDashboard />} />
          <Route path="pokemon" element={<PokemonList />} />
          <Route path="pokemon/new" element={<PokemonForm mode="create" />} />
          <Route path="pokemon/:id/edit" element={<PokemonForm mode="edit" />} />
          <Route path="items" element={<ItemList />} />
          <Route path="items/new" element={<ItemForm mode="create" />} />
          <Route path="items/:id/edit" element={<ItemForm mode="edit" />} />
          <Route path="type-chart" element={<TypeMatchupList />} />
          <Route path="raids" element={<RaidList />} />
          <Route path="raids/new" element={<RaidForm />} />
        </Route>
        <Route path="/api-test" element={<ApiTestPage />} />
        <Route path="/raid-test" element={<RaidTestPage />} />
      </Routes>
    </BrowserRouter>
  );
}
