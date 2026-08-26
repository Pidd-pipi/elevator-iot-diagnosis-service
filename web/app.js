// 前端入口：SPA 路由 + 页面装配。
// 使用 History API 实现无刷新路由，后端对前端路由做 index.html 回退。

import { renderOverview } from './pages/overview.js';
import { renderElevatorDetail } from './pages/elevator-detail.js';
import { renderEvents } from './pages/events.js';
import { renderDiagnosis } from './pages/diagnosis.js';
import { renderWatchlist } from './pages/watchlist.js';

const app = document.getElementById('app');

const routes = [
  { pattern: /^\/$/, render: (container, navigate) => renderOverview(container, navigate) },
  { pattern: /^\/elevators\/([^/]+)$/, render: (container, navigate, match) => renderElevatorDetail(container, navigate, decodeURIComponent(match[1])) },
  { pattern: /^\/events\/([^/]+)$/, render: (container, navigate, match) => renderEventDetail(container, navigate, decodeURIComponent(match[1])) },
  { pattern: /^\/events$/, render: (container, navigate) => renderEvents(container, navigate) },
  { pattern: /^\/diagnosis$/, render: (container, navigate) => renderDiagnosis(container, navigate) },
  { pattern: /^\/watchlist$/, render: (container, navigate) => renderWatchlist(container, navigate) },
];

let cleanup = null;

export function navigate(path) {
  if (window.location.pathname === path) {
    render();
    return;
  }
  history.pushState({}, '', path);
  render();
}

async function render() {
  if (cleanup) { cleanup(); cleanup = null; }
  const path = window.location.pathname;
  app.innerHTML = '<div class="loading">加载中…</div>';
  for (const route of routes) {
    const match = path.match(route.pattern);
    if (!match) continue;
    try {
      const result = await route.render(app, navigate, match);
      if (typeof result === 'function') cleanup = result;
      highlightNav(path);
    } catch (err) {
      app.innerHTML = `<div class="error-box"><h2>页面加载失败</h2><p>${err.message}</p><a href="/" data-internal-link>返回总览</a></div>`;
    }
    return;
  }
  app.innerHTML = `<div class="error-box"><h2>404</h2><p>页面不存在：${path}</p><a href="/" data-internal-link>返回总览</a></div>`;
  highlightNav(path);
}

function highlightNav(path) {
  document.querySelectorAll('#main-nav a').forEach((a) => {
    const nav = a.dataset.nav;
    a.classList.toggle('active', nav === '/' ? path === '/' : path.startsWith(nav));
  });
}

function renderClock() {
  const el = document.getElementById('clock');
  if (!el) return;
  const now = new Date();
  const pad = (n) => String(n).padStart(2, '0');
  el.textContent = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`;
}
setInterval(renderClock, 1000);

// 全局事件委托：内部链接跳转（SPA 路由）。
document.addEventListener('click', (e) => {
  const link = e.target.closest('a[data-internal-link]');
  if (!link) return;
  e.preventDefault();
  navigate(link.getAttribute('href'));
});

// 事件详情页：复用 events 详情接口（含处置轨迹）。
async function renderEventDetail(container, navigate, id) {
  const { api } = await import('./api.js');
  const { eventStatusLabel, fmtTime } = await import('./constants.js');
  const { escapeHtml } = await import('./components/elevator-card.js');
  const data = await api(`/api/events/${encodeURIComponent(id)}`);
  const ev = data.event;
  const disposal = data.disposal;
  const trail = data.trail || [];
  container.innerHTML = `
    <a href="/events" data-internal-link class="back-link">← 返回事件列表</a>
    <h1 class="page-title">事件详情：${escapeHtml(ev.id)}</h1>
    <section class="panel">
      <dl class="kv">
        <dt>电梯</dt><dd>${escapeHtml(ev.elevator_id)}</dd>
        <dt>状态</dt><dd><span class="badge badge-${eventStatusColorFor(ev.status)}">${eventStatusLabel(ev.status)}</span></dd>
        <dt>开始时间</dt><dd>${fmtTime(ev.started_at)}</dd>
        <dt>解除/升级时间</dt><dd>${fmtTime(ev.ended_at)}</dd>
        <dt>升级次数</dt><dd>${ev.escalation_count}${ev.second_alarm_sent ? '（已二次告警）' : ''}</dd>
        <dt>描述</dt><dd>${escapeHtml(ev.description)}</dd>
      </dl>
    </section>
    ${disposal ? `
    <section class="panel">
      <h2 class="panel-title">处置任务</h2>
      <dl class="kv">
        <dt>处置任务</dt><dd>${escapeHtml(disposal.id)}</dd>
        <dt>接单时间</dt><dd>${fmtTime(disposal.accepted_at)}</dd>
        <dt>处置人</dt><dd>${escapeHtml(disposal.disposer || '-')}</dd>
        <dt>处理措施</dt><dd>${escapeHtml(disposal.measure || '-')}</dd>
        <dt>恢复时间</dt><dd>${fmtTime(disposal.recovery_time)}</dd>
        <dt>是否按时</dt><dd>${disposal.timely ? '按时' : '未按时'}</dd>
      </dl>
    </section>` : ''}
    <section class="panel">
      <h2 class="panel-title">处置轨迹（审计）</h2>
      <ul class="timeline">
        ${trail.map((t) => `
          <li><div class="tl-dot"></div><div class="tl-content">
            <div class="tl-head"><span class="tl-code">${escapeHtml(t.action)}</span><span class="muted">${escapeHtml(t.actor)}</span></div>
            <div class="tl-diagnosis">${escapeHtml(t.detail)}</div>
            <div class="tl-time">${fmtTime(t.created_at)}</div>
          </div></li>`).join('') || '<li class="empty">暂无轨迹</li>'}
      </ul>
    </section>
  `;
}

function eventStatusColorFor(status) {
  return { alerted: 'danger', accepted: 'warn', processing: 'info', released: 'success', escalated: 'danger' }[status] || 'info';
}

render();
