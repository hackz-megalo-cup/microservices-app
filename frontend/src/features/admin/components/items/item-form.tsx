import { useQuery } from "@connectrpc/connect-query";
import type { FormEvent } from "react";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { getItem } from "../../../../gen/masterdata/v1/masterdata-MasterdataService_connectquery";
import { useAdminItems } from "../../hooks/use-admin-items";
import type { EffectField } from "./item-effect-fields";
import { BLANK_EFFECT, ItemEffectFields } from "./item-effect-fields";

interface ItemFormProps {
  mode: "create" | "edit";
}

export function ItemForm({ mode }: ItemFormProps) {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();

  const { createMutation, updateMutation } = useAdminItems();

  const itemQuery = useQuery(
    getItem,
    { id: id ?? "" },
    { enabled: mode === "edit" && Boolean(id) },
  );

  const [name, setName] = useState("");
  const [effects, setEffects] = useState<EffectField[]>([]);

  useEffect(() => {
    if (mode === "edit" && itemQuery.data?.item) {
      const item = itemQuery.data.item;
      setName(item.name);
      setEffects(
        item.effects.map((e) => ({
          ...BLANK_EFFECT(),
          effectType: e.effectType,
          targetType: e.targetType,
          captureRateBonus: e.captureRateBonus,
          flavorText: e.flavorText,
        })),
      );
    }
  }, [mode, itemQuery.data]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    const apiEffects = effects.map(({ effectType, targetType, captureRateBonus, flavorText }) => ({
      effectType,
      targetType,
      captureRateBonus,
      flavorText,
    }));
    if (mode === "create") {
      await createMutation.mutateAsync({ name, effects: apiEffects });
    } else {
      await updateMutation.mutateAsync({ id: id ?? "", name, effects: apiEffects });
    }

    void navigate("/admin/items");
  }

  if (mode === "edit" && itemQuery.isPending) {
    return (
      <div className="p-8">
        <p className="text-text-secondary">読み込み中...</p>
      </div>
    );
  }

  if (mode === "edit" && itemQuery.error) {
    return (
      <div className="p-8">
        <p className="text-red-400">エラー: {itemQuery.error.message}</p>
      </div>
    );
  }

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  return (
    <div className="p-8">
      <header className="mb-8">
        <h1 className="text-2xl font-bold text-text-primary">
          {mode === "create" ? "アイテム作成" : "アイテム編集"}
        </h1>
        <p className="text-sm text-text-secondary mt-1">
          {mode === "create" ? "新しいアイテムを作成します" : "アイテムを編集します"}
        </p>
      </header>

      <form onSubmit={(e) => void handleSubmit(e)} className="flex flex-col gap-6 max-w-2xl">
        <div className="bg-bg-card border border-bg-hover rounded-2xl p-6 flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <label htmlFor="item-name" className="text-sm font-medium text-text-secondary">
              名前
            </label>
            <input
              id="item-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-lg text-sm text-text-primary focus:outline-none focus:border-accent"
              placeholder="アイテム名"
            />
          </div>
        </div>

        <div className="bg-bg-card border border-bg-hover rounded-2xl p-6">
          <ItemEffectFields effects={effects} onChange={setEffects} />
        </div>

        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={isSubmitting}
            className="px-6 py-2.5 rounded-xl bg-accent text-bg-primary text-sm font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {isSubmitting ? "保存中..." : "保存"}
          </button>
          <button
            type="button"
            onClick={() => void navigate(-1)}
            className="px-6 py-2.5 rounded-xl bg-bg-hover text-text-secondary text-sm font-medium hover:text-text-primary transition-colors"
          >
            キャンセル
          </button>
        </div>

        {(createMutation.error ?? updateMutation.error) && (
          <p className="text-red-400 text-sm">
            {(createMutation.error ?? updateMutation.error)?.message}
          </p>
        )}
      </form>
    </div>
  );
}
