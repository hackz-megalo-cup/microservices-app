import { useQuery } from "@connectrpc/connect-query";
import { type FormEvent, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import type { Pokemon } from "../../../../gen/masterdata/v1/masterdata_pb";
import { getPokemon } from "../../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { useAdminPokemon } from "../../hooks/use-admin-pokemon";

interface PokemonFormProps {
  mode: "create" | "edit";
  initialData?: Pokemon;
}

interface FormValues {
  name: string;
  type: string;
  hp: string;
  attack: string;
  speed: string;
  specialMoveName: string;
  specialMoveDamage: string;
}

const EMPTY_FORM: FormValues = {
  name: "",
  type: "",
  hp: "",
  attack: "",
  speed: "",
  specialMoveName: "",
  specialMoveDamage: "",
};

function pokemonToFormValues(p: Pokemon): FormValues {
  return {
    name: p.name,
    type: p.type,
    hp: String(p.hp),
    attack: String(p.attack),
    speed: String(p.speed),
    specialMoveName: p.specialMoveName,
    specialMoveDamage: String(p.specialMoveDamage),
  };
}

export function PokemonForm({ mode, initialData }: PokemonFormProps) {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { createMutation, updateMutation } = useAdminPokemon();

  const editQuery = useQuery(getPokemon, { id: id ?? "" }, { enabled: mode === "edit" && !!id });

  const [values, setValues] = useState<FormValues>(() =>
    initialData ? pokemonToFormValues(initialData) : EMPTY_FORM,
  );

  useEffect(() => {
    if (mode === "edit" && editQuery.data?.pokemon) {
      setValues(pokemonToFormValues(editQuery.data.pokemon));
    }
  }, [mode, editQuery.data?.pokemon]);

  function handleChange(field: keyof FormValues, value: string) {
    setValues((prev) => ({ ...prev, [field]: value }));
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();

    const common = {
      name: values.name,
      type: values.type,
      hp: Number(values.hp),
      attack: Number(values.attack),
      speed: Number(values.speed),
      specialMoveName: values.specialMoveName,
      specialMoveDamage: Number(values.specialMoveDamage),
    };

    if (mode === "create") {
      createMutation.mutate(common, {
        onSuccess: () => void navigate("/admin/pokemon"),
      });
    } else {
      updateMutation.mutate(
        { id: id ?? "", ...common },
        { onSuccess: () => void navigate("/admin/pokemon") },
      );
    }
  }

  const isLoading = mode === "edit" && editQuery.isPending;
  const isMutating = createMutation.isPending || updateMutation.isPending;
  const mutationError = createMutation.error ?? updateMutation.error;

  if (isLoading) {
    return (
      <div className="p-8">
        <p className="text-text-secondary">読み込み中...</p>
      </div>
    );
  }

  return (
    <div className="p-8">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-text-primary">
          {mode === "create" ? "Pokemon 新規作成" : "Pokemon 編集"}
        </h1>
      </header>

      <form
        onSubmit={handleSubmit}
        className="bg-bg-card border border-bg-hover rounded-2xl p-6 max-w-lg flex flex-col gap-4"
      >
        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">名前</span>
          <input
            type="text"
            required
            value={values.name}
            onChange={(e) => handleChange("name", e.target.value)}
            className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
          />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">タイプ</span>
          <input
            type="text"
            required
            value={values.type}
            onChange={(e) => handleChange("type", e.target.value)}
            className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
          />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">HP</span>
          <input
            type="number"
            required
            min={1}
            value={values.hp}
            onChange={(e) => handleChange("hp", e.target.value)}
            className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
          />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">Attack</span>
          <input
            type="number"
            required
            min={0}
            value={values.attack}
            onChange={(e) => handleChange("attack", e.target.value)}
            className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
          />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">Speed</span>
          <input
            type="number"
            required
            min={0}
            value={values.speed}
            onChange={(e) => handleChange("speed", e.target.value)}
            className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
          />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">特殊技名</span>
          <input
            type="text"
            required
            value={values.specialMoveName}
            onChange={(e) => handleChange("specialMoveName", e.target.value)}
            className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
          />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-xs font-medium text-text-secondary">特殊技威力</span>
          <input
            type="number"
            required
            min={0}
            value={values.specialMoveDamage}
            onChange={(e) => handleChange("specialMoveDamage", e.target.value)}
            className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-xl text-text-primary text-sm focus:outline-none focus:border-accent"
          />
        </label>

        {mutationError && <p className="text-red-400 text-sm">{mutationError.message}</p>}

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={isMutating}
            className="px-4 py-2 bg-accent text-bg-primary text-sm font-semibold rounded-xl hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {isMutating ? "保存中..." : "保存"}
          </button>
          <button
            type="button"
            onClick={() => void navigate(-1)}
            className="px-4 py-2 bg-bg-hover text-text-primary text-sm font-medium rounded-xl hover:bg-bg-primary transition-colors"
          >
            キャンセル
          </button>
        </div>
      </form>
    </div>
  );
}
