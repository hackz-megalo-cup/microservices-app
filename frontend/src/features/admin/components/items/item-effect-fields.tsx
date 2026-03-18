import type { EffectField } from "../../types";

export type { EffectField } from "../../types";

const EFFECT_TYPES = ["capture_rate_up", "escape"] as const;

export const BLANK_EFFECT = (): EffectField => ({
  _key: crypto.randomUUID(),
  effectType: "",
  targetType: "",
  captureRateBonus: 0,
  flavorText: "",
});

interface ItemEffectFieldsProps {
  effects: EffectField[];
  onChange: (effects: EffectField[]) => void;
}

export function ItemEffectFields({ effects, onChange }: ItemEffectFieldsProps) {
  function handleAdd() {
    onChange([...effects, BLANK_EFFECT()]);
  }

  function handleRemove(key: string) {
    onChange(effects.filter((e) => e._key !== key));
  }

  function handleChange<K extends keyof EffectField>(key: string, field: K, value: EffectField[K]) {
    onChange(effects.map((e) => (e._key === key ? { ...e, [field]: value } : e)));
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold text-text-primary">エフェクト</span>
        <button
          type="button"
          onClick={handleAdd}
          className="px-3 py-1.5 rounded-lg bg-accent text-bg-primary text-sm font-medium hover:opacity-90 transition-opacity"
        >
          エフェクト追加
        </button>
      </div>

      {effects.length === 0 && (
        <p className="text-sm text-text-secondary">エフェクトがありません</p>
      )}

      {effects.map((effect, index) => (
        <div
          key={effect._key}
          className="flex flex-col gap-3 p-4 bg-bg-card border border-bg-hover rounded-xl"
        >
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-text-secondary">エフェクト {index + 1}</span>
            <button
              type="button"
              onClick={() => handleRemove(effect._key)}
              className="px-2 py-1 rounded-lg bg-bg-hover text-text-secondary text-xs hover:text-text-primary transition-colors"
            >
              削除
            </button>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1">
              <label htmlFor={`${effect._key}-effectType`} className="text-xs text-text-secondary">
                エフェクトタイプ
              </label>
              <select
                id={`${effect._key}-effectType`}
                value={effect.effectType}
                onChange={(e) => handleChange(effect._key, "effectType", e.target.value)}
                className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-lg text-sm text-text-primary focus:outline-none focus:border-accent"
              >
                <option value="">選択してください</option>
                {EFFECT_TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex flex-col gap-1">
              <label htmlFor={`${effect._key}-targetType`} className="text-xs text-text-secondary">
                ターゲットタイプ
              </label>
              <input
                id={`${effect._key}-targetType`}
                type="text"
                value={effect.targetType}
                onChange={(e) => handleChange(effect._key, "targetType", e.target.value)}
                className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-lg text-sm text-text-primary focus:outline-none focus:border-accent"
                placeholder="例: fire"
              />
            </div>

            <div className="flex flex-col gap-1">
              <label
                htmlFor={`${effect._key}-captureRateBonus`}
                className="text-xs text-text-secondary"
              >
                捕獲率ボーナス
              </label>
              <input
                id={`${effect._key}-captureRateBonus`}
                type="number"
                value={effect.captureRateBonus}
                onChange={(e) =>
                  handleChange(effect._key, "captureRateBonus", Number(e.target.value))
                }
                className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-lg text-sm text-text-primary focus:outline-none focus:border-accent"
              />
            </div>

            <div className="flex flex-col gap-1">
              <label htmlFor={`${effect._key}-flavorText`} className="text-xs text-text-secondary">
                フレーバーテキスト
              </label>
              <input
                id={`${effect._key}-flavorText`}
                type="text"
                value={effect.flavorText}
                onChange={(e) => handleChange(effect._key, "flavorText", e.target.value)}
                className="px-3 py-2 bg-bg-primary border border-bg-hover rounded-lg text-sm text-text-primary focus:outline-none focus:border-accent"
                placeholder="説明テキスト"
              />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
