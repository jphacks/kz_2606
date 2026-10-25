import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import Result from './result';

describe('Result', () => {
  it('renders persisted activity values instead of demo constants', () => {
    const markup = renderToStaticMarkup(
      <Result
        username="walker"
        history={{
          summary: { recordings: 1, completed: 1, meters: 1500, elapsedMs: 605_000 },
          activities: [{
            routeId: 12,
            roomCode: 'ABC234',
            roomTitle: '実測したルート',
            isPrivate: false,
            memberCount: 2,
            startedAt: '2026-10-24T05:00:00Z',
            endedAt: '2026-10-24T05:10:05Z',
            meters: 1500,
            elapsedMs: 605_000,
          }],
        }}
      />,
    );

    expect(markup).toContain('1.50 km');
    expect(markup).toContain('10分5秒');
    expect(markup).toContain('実測したルート');
    expect(markup).not.toContain('4.80');
    expect(markup).not.toContain('2時間24分');
  });

  it('shows an empty state for users without routes', () => {
    const markup = renderToStaticMarkup(
      <Result
        history={{
          summary: { recordings: 0, completed: 0, meters: 0, elapsedMs: 0 },
          activities: [],
        }}
      />,
    );

    expect(markup).toContain('まだ記録がありません');
    expect(markup).toContain('0 m');
    expect(markup).toContain('0秒');
  });
});
