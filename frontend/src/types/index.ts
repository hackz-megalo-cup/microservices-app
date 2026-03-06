/** API のベース URL */
export type { Transport } from '@connectrpc/connect';

/** JSON レスポンスの汎用型 */
export interface JsonResponse {
  [key: string]: unknown;
}
