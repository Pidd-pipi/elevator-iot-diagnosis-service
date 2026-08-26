// 电梯总览页（/）。
// 数据来源：GET /api/overview + GET /api/elevators + GET /api/events。
// 共享组件：ElevatorCard、EventTable；共享 hook：useElevators、useEvents。

import { api } from '../api.js';
import { useElevators } from '../hooks/use-elevators.js';
import { useEvents } from '../hooks/use-events.js';
import { ElevatorCard, escapeHtml } from '../components/elevator-card.js';
import { EventTable } from '../components/event-table.js';
import { fmtTime } from '../constants.js';

export async function renderOverview(container, navigate) {
  const statsHook = useElevators(true);
  const eventsHook = useEvents();
  const [stats] = await Promise.all([api('/api/overview'), statsHook.reload(), eventsHook.reload()]);
  statsHook.startPolling();

  container.innerHTML = `
    <h1 class="page-title">电梯总览</h1>
    <div class="stats-grid">
      <div class="stat-card"><div class="stat-num">${stats.stats.elevator_total}</div><div class="stat-label">电梯总数</div></div>
      <div class="stat-card warn"><div class="stat-num">${stats.stats.open_events}</div><div class="stat-label">进行中困人</div></div>
      <div class="stat-card"><div class="stat-num">${stats.stats.reports_today}</div><div class="stat-label">今日上报</div></div>
      <div class="stat-card danger"><div class="stat-num">${stats.stats.unknown_faults}</div><div class="stat-label">未知故障码</div></div>
      <div class="stat-card"><div class="stat-num">${stats.stats.watchlist_total}</div><div class="stat-label">重点关注</div></div>
      <div class="stat-card"><div class="stat-num">${stats.stats.released_total}</div><div class="stat-label">累计已解除</div></div>
    </div>

    <section class="panel">
      <h2 class="panel-title">进行中的困人事件</h2>
      <div id="overview-events"></div>
    </section>

    <section class="panel">
      <div class="panel-head-row">
        <h2 class="panel-title">电梯状态</h2>
        <span class="muted">每 5 秒自动刷新</span>
      </div>
      <div id="overview-elevators" class="card-grid"></div>
    </section>

    <section class="panel">
      <h2 class="panel-title">最近状态上报</h2>
      <div id="overview-reports"></div>
    </section>
  `;

  const eventsBox = container.querySelector('#overview-events');
  const openEvents = stats.open_events || [];
  eventsBox.appendChild(EventTable(openEvents, {
    actions: [
      { name: 'accept', label: '接单', kind: 'primary', handler: async (id) => { await handleAccept(id, eventsHook); } },
      { name: 'detail', label: '详情', kind: 'info', handler: (id) => navigate(`/events/${id}`) },
    ],
  }));

  const elevBox = container.querySelector('#overview-elevators');
  for (const e of statsHook.elevators) {
    elevBox.appendChild(ElevatorCard(e, { onClick: () => navigate(`/elevators/${encodeURIComponent(e.id)}`) }));
  }

  const reportBox = container.querySelector('#overview-reports');
  const table = document.createElement('table');
  table.className = 'event-table';
  table.innerHTML = `<thead><tr><th>电梯</th><th>楼层</th><th>方向</th><th>门</th><th>平层</th><th>故障码</th><th>时间</th></tr></thead>`;
  const tbody = document.createElement('tbody');
  for (const r of stats.recent_reports || []) {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td>${escapeHtml(r.elevator_id)}</td>
      <td>${r.floor}</td>
      <td>${escapeHtml(r.direction)}</td>
      <td>${escapeHtml(r.door)}</td>
      <td>${r.leveling ? '是' : '否'}</td>
      <td>${escapeHtml(r.fault_code || '-')}</td>
      <td>${fmtTime(r.reported_at)}</td>`;
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);
  reportBox.appendChild(table);

  return () => statsHook.stopPolling();
}

async function handleAccept(id, eventsHook) {
  try {
    await api(`/api/events/${id}/accept`, { method: 'POST' });
    toast('接单成功');
    await eventsHook.reload();
  } catch (err) {
    toast('接单失败：' + err.message, true);
  }
}

export function toast(msg, isError = false) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast ' + (isError ? 'toast-error' : 'toast-success');
  el.hidden = false;
  clearTimeout(window.__toastTimer);
  window.__toastTimer = setTimeout(() => { el.hidden = true; }, 3000);
}
