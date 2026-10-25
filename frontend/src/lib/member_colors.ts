export const memberColors = [
  { name: 'グリーン', hex: '#0e7861' },
  { name: 'オレンジ', hex: '#e67e22' },
  { name: 'ブルー', hex: '#3498db' },
  { name: 'ピンク', hex: '#e91e63' },
  { name: 'パープル', hex: '#9b59b6' },
  { name: 'イエロー', hex: '#f1c40f' },
  { name: 'レッド', hex: '#e74c3c' },
  { name: 'ターコイズ', hex: '#1abc9c' },
  { name: 'インディゴ', hex: '#3f51b5' },
  { name: 'ライム', hex: '#8bc34a' },
] as const;

export const defaultMemberColor = memberColors[0].hex;

/** バックエンドの NormalizeColor と同じ表現へ揃える。 */
export function normalizeMemberColor(color: string): string {
  return color.toLowerCase();
}
