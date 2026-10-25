export const memberColors = [
  { name: 'グリーン', hex: '#0E7861' },
  { name: 'オレンジ', hex: '#E67E22' },
  { name: 'ブルー', hex: '#3498DB' },
  { name: 'ピンク', hex: '#E91E63' },
  { name: 'パープル', hex: '#9B59B6' },
  { name: 'イエロー', hex: '#F1C40F' },
  { name: 'レッド', hex: '#E74C3C' },
  { name: 'ターコイズ', hex: '#1ABC9C' },
  { name: 'インディゴ', hex: '#3F51B5' },
  { name: 'ライム', hex: '#8BC34A' },
] as const;

export const defaultMemberColor = memberColors[0].hex;
