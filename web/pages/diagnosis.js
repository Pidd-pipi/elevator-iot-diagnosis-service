// 故障诊断页（/diagnosis）。
// 数据来源：GET /api/diagnosis（故障码映射表 + 未知故障码）。
// 共享组件：FaultTimeline（展示未知故障码记录）。

import { api } from '../api.js';
import { FaultTimeline } from '../components/fault-timeline.js';
import { SEVERITY } from '../constants.js';
import { escapeHtml } from '../components/elevator-card.js';

export async function renderDiagnosis(container, navigate) {
  const data = await api('/api/diagnosis');
  const rules = data.rules || [];
  const unknown = data.unknown || [];

  container.innerHTML = `
    <h1 class="page-title">故障诊断</h1>
    <div class="stats-grid small">
      <div class="stat-card"><div class="stat-num">${rules.length}</div><div class="stat-label">已收录故障码</div></div>
      <div class="stat-card danger"><div class="stat-num">${data.unknown_cnt}</div><div class="stat-label">未知故障码待人工确认</div></div>
    </div>

    <section class="panel">
      <h2 class="panel-title">故障码诊断映射表</h2>
      <table class="event-table">
        <thead><tr><th>故障码</th><th>名称</th><th>类别</th><th>严重度</th><th>诊断结论</th><th>处理建议</th></tr></thead>
        <tbody>
          ${rules.map((r) => `
            <tr>
              <td><strong>${escapeHtml(r.code)}</strong></td>
              <td>${escapeHtml(r.name)}</td>
              <td>${escapeHtml(r.category)}</td>
              <td><span class="badge badge-${r.severity === 'high' ? 'danger' : r.severity === 'medium' ? 'warn' : 'info'}">${SEVERITY[r.severity] || r.severity}</span></td>
              <td>${escapeHtml(r.diagnosis)}</td>
              <td>${escapeHtml(r.suggestion)}</td>
            </tr>`).join('')}
        </tbody>
      </table>
    </section>

    <section class="panel">
      <h2 class="panel-title">未知故障码记录（需人工确认）</h2>
      <div id="unknown-timeline"></div>
    </section>
  `;

  const unknownFaults = unknown.map((f) => ({ ...f, severity: 'high' }));
  container.querySelector('#unknown-timeline').appendChild(FaultTimeline(unknownFaults, {
    showElevator: true,
    emptyText: '暂无未知故障码，知识库覆盖良好',
  }));
}
