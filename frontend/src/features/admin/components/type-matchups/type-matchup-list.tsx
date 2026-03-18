import { type KeyboardEvent, useState } from "react";
import { useAdminTypeMatchups } from "../../hooks/use-admin-type-matchups";
import { TypeMatchupForm } from "./type-matchup-form";

interface EditingCell {
  attackingType: string;
  defendingType: string;
  value: string;
}

export function TypeMatchupList() {
  const { matchups, isLoading, error, updateMutation, deleteMutation } = useAdminTypeMatchups();
  const [editingCell, setEditingCell] = useState<EditingCell | null>(null);

  function handleEffectivenessClick(attackingType: string, defendingType: string, current: number) {
    setEditingCell({ attackingType, defendingType, value: String(current) });
  }

  function handleEffectivenessCommit() {
    if (!editingCell) {
      return;
    }
    const effectiveness = Number(editingCell.value);
    if (!Number.isNaN(effectiveness)) {
      updateMutation.mutate(
        {
          attackingType: editingCell.attackingType,
          defendingType: editingCell.defendingType,
          effectiveness,
        },
        { onSuccess: () => setEditingCell(null) },
      );
    } else {
      setEditingCell(null);
    }
  }

  function handleEffectivenessKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      handleEffectivenessCommit();
    } else if (e.key === "Escape") {
      setEditingCell(null);
    }
  }

  function handleDelete(attackingType: string, defendingType: string) {
    if (!window.confirm(`「${attackingType} → ${defendingType}」の相性を削除しますか？`)) {
      return;
    }
    deleteMutation.mutate({ attackingType, defendingType });
  }

  if (isLoading) {
    return (
      <div className="p-8">
        <p className="text-text-secondary">読み込み中...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8">
        <p className="text-red-400">エラー: {error.message}</p>
      </div>
    );
  }

  return (
    <div className="p-8">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-text-primary">タイプ相性</h1>
        <p className="text-sm text-text-secondary mt-1">タイプ相性テーブルの管理</p>
      </header>

      <div className="bg-bg-card border border-bg-hover rounded-2xl overflow-hidden mb-8">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-bg-hover">
              <th className="px-4 py-3 text-left text-text-secondary font-medium">攻撃タイプ</th>
              <th className="px-4 py-3 text-left text-text-secondary font-medium">防御タイプ</th>
              <th className="px-4 py-3 text-right text-text-secondary font-medium">倍率</th>
              <th className="px-4 py-3 text-right text-text-secondary font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            {matchups.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-text-secondary">
                  データがありません
                </td>
              </tr>
            ) : (
              matchups.map((m) => {
                const isEditing =
                  editingCell?.attackingType === m.attackingType &&
                  editingCell?.defendingType === m.defendingType;
                return (
                  <tr
                    key={`${m.attackingType}-${m.defendingType}`}
                    className="border-b border-bg-hover last:border-0 hover:bg-bg-hover transition-colors"
                  >
                    <td className="px-4 py-3 text-text-primary">{m.attackingType}</td>
                    <td className="px-4 py-3 text-text-primary">{m.defendingType}</td>
                    <td className="px-4 py-3 text-right">
                      {isEditing ? (
                        <input
                          type="number"
                          step="0.25"
                          value={editingCell.value}
                          onChange={(e) =>
                            setEditingCell((prev) => prev && { ...prev, value: e.target.value })
                          }
                          onBlur={handleEffectivenessCommit}
                          onKeyDown={handleEffectivenessKeyDown}
                          className="w-24 px-2 py-1 bg-bg-primary border border-accent rounded-lg text-text-primary text-sm text-right focus:outline-none"
                        />
                      ) : (
                        <button
                          type="button"
                          onClick={() =>
                            handleEffectivenessClick(
                              m.attackingType,
                              m.defendingType,
                              m.effectiveness,
                            )
                          }
                          className="text-text-primary hover:text-accent transition-colors cursor-pointer"
                          title="クリックして編集"
                        >
                          {m.effectiveness}
                        </button>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        type="button"
                        onClick={() => handleDelete(m.attackingType, m.defendingType)}
                        disabled={deleteMutation.isPending}
                        className="px-3 py-1 text-xs font-medium text-red-400 bg-bg-hover rounded-lg hover:bg-bg-primary transition-colors disabled:opacity-50"
                      >
                        削除
                      </button>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      <section>
        <h2 className="text-lg font-semibold text-text-primary mb-4">新規作成</h2>
        <TypeMatchupForm />
      </section>
    </div>
  );
}
