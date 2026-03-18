import { useState } from "react";
import { useAuthContext } from "../../../lib/auth";
import "../../../styles/global.css";
import { useCaptureItems } from "../hooks/use-capture-items";
import { NavBar } from "./ui/nav-bar";

export function Capture() {
  const { user } = useAuthContext();
  const userId = user?.id ?? "";
  const [showItemModal, setShowItemModal] = useState(false);
  const [captureRateBonus, setCaptureRateBonus] = useState(0);

  const {
    availableItems,
    isLoading,
    error,
    handleUseItem: handleUseItemApi,
    isPending,
    refetch,
  } = useCaptureItems(userId);

  const baseRate = 0.42;
  const currentRate = Math.min(baseRate + captureRateBonus, 1);

  const handleUseItem = (itemId: string, bonus: number) => {
    handleUseItemApi(itemId, bonus);
    setCaptureRateBonus(bonus);
    setShowItemModal(false);
  };

  if (isLoading) {
    return (
      <div className="showcase-screen">
        <NavBar title="CAPTURE" />
        <div className="flex-1 flex items-center justify-center text-text-secondary text-sm">
          Loading...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="showcase-screen">
        <NavBar title="CAPTURE" />
        <div className="flex-1 flex flex-col items-center justify-center gap-3 px-6">
          <p className="text-sm text-text-secondary m-0">{error.message}</p>
          <button
            type="button"
            className="bg-bg-card text-text-primary rounded-full px-4 py-2 border-none cursor-pointer"
            onClick={() => void refetch()}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }
  return (
    <div className="showcase-screen">
      <NavBar title="CAPTURE" />

      <div className="flex-1 flex flex-col items-center justify-center gap-6 px-6">
        <img
          src="/images/capture-python.png"
          alt="Python"
          className="w-[200px] h-[200px] rounded-full object-cover"
        />
        <div className="flex flex-col items-center gap-1">
          <span className="text-2xl font-bold text-text-primary">Python</span>
          <span className="text-5xl font-bold text-accent">{Math.round(currentRate * 100)}%</span>
        </div>
        <button
          type="button"
          className="w-20 h-20 rounded-full bg-accent border-none text-[32px] cursor-pointer flex items-center justify-center hover:opacity-90"
        >
          🎯
        </button>
        <span className="text-sm text-text-secondary">tap to throw</span>
      </div>

      <div className="flex gap-3 w-full px-6 pb-6">
        <button
          type="button"
          className="flex-1 flex items-center justify-center gap-2 px-5 py-4 bg-bg-card rounded-2xl border-none text-sm font-bold text-text-primary cursor-pointer hover:bg-bg-hover"
          onClick={() => setShowItemModal(true)}
          disabled={isPending}
        >
          Use Item
        </button>
        <button
          type="button"
          className="flex-1 flex items-center justify-center gap-2 px-5 py-4 bg-bg-card rounded-2xl border-none text-sm font-bold text-text-secondary cursor-pointer hover:bg-bg-hover"
        >
          Skip
        </button>
      </div>

      {showItemModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-bg-primary rounded-3xl p-6 max-w-sm w-[90%] max-h-[80vh] flex flex-col">
            <h2 className="text-xl font-bold text-text-primary mb-4">Select Item</h2>

            {availableItems.length === 0 && (
              <p className="text-sm text-text-secondary text-center flex-1 flex items-center justify-center">
                No items available
              </p>
            )}

            <div className="flex-1 overflow-y-auto space-y-2">
              {availableItems.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className="w-full bg-bg-card rounded-2xl p-4 text-left border-none cursor-pointer hover:bg-bg-hover transition"
                  onClick={() => handleUseItem(item.id, item.captureRateBonus)}
                  disabled={isPending}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <p className="text-sm font-bold text-text-primary m-0">{item.name}</p>
                      <p className="text-xs text-text-secondary m-0">
                        Capture Rate +{Math.round(item.captureRateBonus * 100)}%
                      </p>
                    </div>
                    <span className="text-xs font-bold text-text-secondary bg-bg-primary px-2 py-1 rounded">
                      x{item.quantity}
                    </span>
                  </div>
                </button>
              ))}
            </div>

            <button
              type="button"
              className="mt-4 w-full bg-bg-card rounded-2xl px-4 py-2 border-none text-sm font-bold text-text-secondary cursor-pointer hover:bg-bg-hover"
              onClick={() => setShowItemModal(false)}
              disabled={isPending}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
