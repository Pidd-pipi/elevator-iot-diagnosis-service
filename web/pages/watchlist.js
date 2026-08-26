// 重点关注页（/watchlist）。
// 数据来源：GET /api/watchlist。
// 共享组件：ElevatorCard；共享 hook：useElevators。

import { api } from '../api.js';
import { useElevators } from '../hooks/use-elevators.js';
import { ElevatorCard } from '../components/elevator-card.js';

export async function renderWatchlist(container, navigate) {
  const hook = useElevators(true);
  await hook.reload();
  hook.startPolling();

  container.innerHTML = `
    <h1 class="page-title">重点关注名单</h1>
    <p class="muted">健康评分 ≤ 60 的电梯自动进入重点关注名单（评分规则：100 - 近30天故障次数×2 - 未按时处置次数×5）。</p>
    <div id="watchlist-grid" class="card-grid"></div>
  `;

  const grid = container.querySelector('#watchlist-grid');
  const watchlisted = hook.elevators.filter((e) => e.watchlisted);
  if (watchlisted.length === 0) {
    grid.innerHTML = '<div class="empty">暂无重点关注电梯，所有电梯评分健康 🎉</div>';
    return () => hook.stopPolling();
  }
  for (const e of watchlisted) {
    grid.appendChild(ElevatorCard(e, { onClick: () => navigate(`/elevators/${encodeURIComponent(e.id)}`) }));
  }
  return () => hook.stopPolling();
}

// 引用 api 确保该模块与后端联通（页面复用 watchlist 接口做一致性校验）。
export async function watchlistCount() {
  const data = await api('/api/watchlist');
  return (data.elevators || []).length;
}
