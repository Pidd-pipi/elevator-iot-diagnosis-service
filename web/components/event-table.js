// EventTable 共享组件：困人事件表格。
// 被「电梯总览」与「困人事件」页面共用。

import { eventStatusColor, eventStatusLabel, fmtTime, PASSENGER_SIGNAL } from '../constants.js';
import { escapeHtml } from './elevator-card.js';

export function EventTable(events, { actions = [], emptyText = '暂无困人事件' } = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'event-table-wrap';

  if (!events || events.length === 0) {
    wrap.innerHTML = `<div class="empty">${escapeHtml(emptyText)}</div>`;
    return wrap;
  }

  const table = document.createElement('table');
  table.className = 'event-table';
  const head = document.createElement('thead');
  head.innerHTML = `
    <tr>
      <th>事件 ID</th><th>电梯</th><th>状态</th><th>乘客信号</th>
      <th>开始时间</th><th>已持续/时长</th><th>升级</th>${actions.length ? '<th>操作</th>' : ''}
    </tr>`;
  const body = document.createElement('tbody');

  for (const ev of events) {
    const tr = document.createElement('tr');
    tr.dataset.eventId = ev.id;
    const duration = ev.duration_sec ? `${ev.duration_sec}s` : (ev.started_at ? fmtTime(ev.started_at) : '-');
    let actionCells = '';
    if (actions.length) {
      actionCells = `<td class="actions">${actions.map((a) => `
        <button class="btn btn-sm btn-${a.kind || 'primary'}" data-action="${a.name}" data-event-id="${escapeHtml(ev.id)}">${escapeHtml(a.label)}</button>`).join('')}</td>`;
    }
    tr.innerHTML = `
      <td><a href="/events/${escapeHtml(ev.id)}" data-internal-link>${escapeHtml(ev.id)}</a></td>
      <td>${escapeHtml(ev.elevator_id)}</td>
      <td><span class="badge badge-${eventStatusColor(ev.status)}">${eventStatusLabel(ev.status)}</span></td>
      <td>${escapeHtml(PASSENGER_SIGNAL[ev.passenger_signal] || ev.passenger_signal || '-')}</td>
      <td>${fmtTime(ev.started_at)}</td>
      <td>${escapeHtml(duration)}</td>
      <td>${ev.escalation_count || 0}${ev.second_alarm_sent ? '（二次告警）' : ''}</td>
      ${actionCells}`;
    body.appendChild(tr);
  }
  table.appendChild(head);
  table.appendChild(body);
  wrap.appendChild(table);

  // 操作按钮事件委托。
  wrap.addEventListener('click', (e) => {
    const btn = e.target.closest('button[data-action]');
    if (!btn) return;
    const action = actions.find((a) => a.name === btn.dataset.action);
    if (action && action.handler) {
      action.handler(btn.dataset.eventId, btn);
    }
  });
  return wrap;
}
