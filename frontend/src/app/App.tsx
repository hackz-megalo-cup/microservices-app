import { BrowserRouter, Route, Routes } from "react-router";
import { AdminDashboard, AdminLayout } from "../features/admin";
import { ItemForm } from "../features/admin/components/items/item-form";
import { ItemList } from "../features/admin/components/items/item-list";
import { PokemonForm } from "../features/admin/components/pokemon/pokemon-form";
import { PokemonList } from "../features/admin/components/pokemon/pokemon-list";
import { RaidForm } from "../features/admin/components/raids/raid-form";
import { RaidList } from "../features/admin/components/raids/raid-list";
import { TypeMatchupList } from "../features/admin/components/type-matchups/type-matchup-list";
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
import { Profile } from "../features/showcase/components/profile";
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
