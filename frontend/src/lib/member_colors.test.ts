import { describe, expect, it } from 'vitest';
import { defaultMemberColor, memberColors, normalizeMemberColor } from './member_colors';

describe('member colors', () => {
  it('uses the existing green-first palette in the server canonical form', () => {
    expect(defaultMemberColor).toBe('#0e7861');
    expect(memberColors.map((color) => color.hex)).toEqual([
      '#0e7861',
      '#e67e22',
      '#3498db',
      '#e91e63',
      '#9b59b6',
      '#f1c40f',
      '#e74c3c',
      '#1abc9c',
      '#3f51b5',
      '#8bc34a',
    ]);
    for (const color of memberColors) {
      expect(color.hex).toMatch(/^#[0-9a-f]{6}$/);
      expect(normalizeMemberColor(color.hex)).toBe(color.hex);
    }
  });

  it('normalizes colors returned with another letter case', () => {
    expect(normalizeMemberColor('#0E7861')).toBe(defaultMemberColor);
  });
});
