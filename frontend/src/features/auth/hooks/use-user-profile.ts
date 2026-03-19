import { useQuery } from "@connectrpc/connect-query";
import { getUserProfile } from "../../../gen/auth/v1/auth-AuthService_connectquery";
import type { UserProfileResult } from "../types";

export function useUserProfile(userId: string): UserProfileResult {
  const query = useQuery(getUserProfile, { userId }, { enabled: !!userId });

  const user = query.data?.user;

  const createdAt = user?.createdAt
    ? new Date(Number(user.createdAt.seconds) * 1000).toLocaleDateString("ja-JP")
    : null;

  const error =
    query.error instanceof Error
      ? query.error
      : query.error
        ? new Error("プロフィール取得失敗")
        : null;

  return {
    profile: user
      ? {
          id: user.id,
          email: user.email,
          name: user.name,
          role: user.role,
          createdAt,
        }
      : null,
    isLoading: query.isPending,
    error,
  };
}
