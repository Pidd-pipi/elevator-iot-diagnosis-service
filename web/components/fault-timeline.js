// FaultTimeline 共享组件：故障码时间线。
// 被「电梯详情」与「故障诊断」页面共用。

import { fmtTime, FAULT_TYPE, SEVERITY } from '../constants.js';
import { escapeHtml } from './elevator-card.js';

export function FaultTimeline(faults, { showElevator = false, emptyText = '暂无故障记录' } = {}) {
  const wrap = document.createElement('div');
  wrap.className = 'fault-timeline';

  if (!faults || faults.length === 0) {
    wrap.innerHTML = `<div class="empty">${escapeHtml(emptyText)}</div>`;
    return wrap;
  }

  const list = document.createElement('ul');
  list.className = 'timeline';
  for (const f of faults) {
    const li = document.createElement('li');
    li.className = f.known ? 'tl-known' : 'tl-unknown';
    const sev = SEVERITY[f.severity] || f.severity || '-';
    li.innerHTML = `
      <div class="tl-dot ${f.known ? '' : 'dot-unknown'}"></div>
      <div class="tl-content">
        <div class="tl-head">
          <span class="tl-code">${escapeHtml(f.fault_code || f.code)}</span>
          ${showElevator ? `<span class="tl-elevator">${escapeHtml(f.elevator_id)}</span>` : ''}
          <span class="badge ${f.known ? 'badge-info' : 'badge-danger'}">${FAULT_TYPE[f.fault_type] || (f.known ? '已知故障' : '未知故障')}</span>
        </div>
        <div class="tl-diagnosis">${escapeHtml(f.diagnosis)}</div>
        <div class="tl-suggestion">建议：${escapeHtml(f.suggestion)}</div>
        <div class="tl-time">${fmtTime(f.occurred_at)}</div>
      </div>`;
    list.appendChild(li);
  }
  wrap.appendChild(list);
  return wrap;
}
