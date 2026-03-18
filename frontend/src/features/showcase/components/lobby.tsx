import { useMutation } from "@connectrpc/connect-query";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";
import {
  joinRaid,
  startBattle,
} from "../../../gen/raid_lobby/v1/raid_lobby-RaidLobbyService_connectquery";
import { getCurrentUserId } from "../../../lib/auth";
import { transport } from "../../../lib/transport";
import { useLobbyStream } from "../hooks/use-lobby-stream";
import "../styles/global.css";
import { NavBar } from "./ui/nav-bar";

export function Lobby() {
  const { id: lobbyId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [hasJoined, setHasJoined] = useState(false);
  const userId = useMemo(() => getCurrentUserId(), []);

  // --- 1. JoinRaid (Unary) ---
  const joinMutation = useMutation(joinRaid, { transport });

  useEffect(() => {
    if (!lobbyId || hasJoined) {
      return;
    }

    joinMutation
      .mutateAsync({ lobbyId, userId })
      .then(() => {
        setHasJoined(true);
        console.log("[JoinRaid] 成功");
      })
      .catch((err) => {
        console.error("[JoinRaid] 失敗:", err);
      });
  }, [lobbyId, hasJoined, userId, joinMutation]);

  // --- 2. StreamLobby (Server Streaming) ---
  const {
    participants,
    isConnected,
    error: streamError,
    battleSessionId,
  } = useLobbyStream(lobbyId ?? "");

  // --- 3. battle_started 受信時の自動遷移 ---
  useEffect(() => {
    if (battleSessionId) {
      navigate(`/battle/${battleSessionId}`);
    }
  }, [battleSessionId, navigate]);

  // --- 4. StartBattle (Unary) ---
  const startMutation = useMutation(startBattle, { transport });

  const handleStartBattle = async () => {
    if (!lobbyId) {
      return;
    }
    try {
      const res = await startMutation.mutateAsync({ lobbyId });
      navigate(`/battle/${res.battleSessionId}`);
    } catch (err) {
      console.error("[StartBattle] 失敗:", err);
    }
  };

  // --- 5. ローディング・エラー表示 ---
  if (joinMutation.isPending) {
    return (
      <div className="showcase-screen flex items-center justify-center">
        <p className="text-text-primary">ロビーに参加中...</p>
      </div>
    );
  }

  if (joinMutation.isError) {
    return (
      <div className="showcase-screen flex items-center justify-center flex-col gap-3">
        <p className="text-text-primary">ロビーへの参加に失敗しました</p>
        <p className="text-text-secondary text-sm">{joinMutation.error?.message}</p>
      </div>
    );
  }

  return (
    <div className="showcase-screen">
      <NavBar title="RAID LOBBY" rightIcon="share" />

      <div className="flex flex-col gap-3 px-4 pb-4 flex-1">
        <section className="flex flex-col items-center gap-3 py-2">
          <img
            src="/images/lobby-python.png"
            alt="Python"
            className="w-[160px] h-[160px] rounded-full object-cover"
          />
          <div className="flex items-center gap-2">
            <span className="text-xl font-bold text-text-primary">Python</span>
            <span className="text-xs text-accent bg-bg-card px-3 py-1 rounded-lg">Dynamic</span>
          </div>
        </section>

        <div className="flex flex-col items-center gap-3 bg-bg-card rounded-2xl p-6">
          <div className="w-20 h-20 flex items-center justify-center text-2xl font-bold text-accent">
            QR
          </div>
          <div className="flex items-center gap-2">
            <span className="text-lg font-bold text-text-primary">
              {lobbyId?.substring(0, 8).toUpperCase()}
            </span>
            <button
              type="button"
              className="bg-transparent text-base cursor-pointer p-1"
              aria-label="Copy code"
              onClick={() => navigator.clipboard.writeText(lobbyId ?? "")}
            >
              📋
            </button>
          </div>
        </div>

        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-bold tracking-widest text-text-secondary">TRAINERS</span>
            <span className="text-xs font-bold tracking-widest text-text-secondary">
              {participants.length}/6
            </span>
          </div>

          {/* 接続状態インジケーター */}
          <div className="flex items-center gap-2 text-xs text-text-secondary">
            <div
              className={`w-2 h-2 rounded-full ${isConnected ? "bg-accent" : "bg-text-secondary"}`}
            />
            <span>{isConnected ? "リアルタイム接続中" : "切断"}</span>
          </div>

          {participants.length === 0 ? (
            <div className="flex items-center justify-center py-6">
              <p className="text-text-secondary text-sm">参加者を待っています...</p>
            </div>
          ) : (
            participants.map((participant) => (
              <div
                key={participant.id}
                className="flex items-center gap-3 bg-bg-card rounded-2xl px-4 py-3"
              >
                <div className="w-8 h-8 rounded-full bg-bg-hover shrink-0" />
                <span className="text-sm text-text-primary flex-1">{participant.name}</span>
                <span className="text-xs text-text-secondary">{participant.pokemon}</span>
                <div
                  className={`w-2 h-2 rounded-full shrink-0 ${participant.online ? "bg-accent" : "bg-text-secondary"}`}
                />
              </div>
            ))
          )}
        </section>

        {streamError && (
          <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3">
            <p className="text-xs text-red-500">ストリーム接続エラー: {streamError.message}</p>
          </div>
        )}

        <div className="flex-1" />

        <button
          type="button"
          onClick={handleStartBattle}
          disabled={startMutation.isPending || participants.length === 0}
          className="w-full h-14 bg-accent rounded-3xl text-base font-bold text-bg-primary cursor-pointer hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {startMutation.isPending ? "バトル開始中..." : "ENTER RAID"}
        </button>
      </div>
    </div>
  );
}
