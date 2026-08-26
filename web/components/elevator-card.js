// ElevatorCard 共享组件：电梯状态卡片。
// 被「电梯总览」与「重点关注」页面共用。

import { fmtTime } from '../constants.js';

export function ElevatorCard(elevator, { onClick, compact = false } = {}) {
  const scoreClass = elevator.health_score <= 60 ? 'score-bad' : elevator.health_score <= 80 ? 'score-mid' : 'score-good';
  const statusMap = {
    normal: '运行正常',
    fault: '故障告警',
    trapped: '困人告警',
    unknown: '暂无数据',
  };
  const statusText = statusMap[elevator.status] || elevator.status || '暂无数据';
  const statusClass = elevator.status === 'trapped' ? 'danger' : elevator.status === 'fault' ? 'warn' : 'normal';

  const el = document.createElement('div');
  el.className = `elevator-card ${compact ? 'compact' : ''}`;
  el.innerHTML = `
    <div class="card-head">
      <span class="elevator-id">${escapeHtml(elevator.id)}</span>
      ${elevator.watchlisted ? '<span class="badge badge-danger">重点关注</span>' : ''}
    </div>
    <div class="card-body">
      <div class="card-meta">${escapeHtml(elevator.building)} · ${escapeHtml(elevator.model)}</div>
      <div class="score-row">
        <span class="score-badge ${scoreClass}">${elevator.health_score}</span>
        <span>健康评分</span>
      </div>
      <div class="status-row status-${statusClass}">● ${statusText}</div>
      <div class="card-foot">最近上报：${fmtTime(elevator.last_report_at)}</div>
    </div>
  `;
  if (onClick) {
    el.classList.add('clickable');
    el.addEventListener('click', onClick);
  }
  return el;
}

export function escapeHtml(str) {
  return String(str ?? '').replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[c]));
}
