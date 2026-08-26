// 电梯详情页（/elevators/{id}）。
// 数据来源：GET /api/elevators/{id}、/score、/faults、/events?elevator_id=
// 共享组件：FaultTimeline、EventTable；并提供一个「模拟状态上报」表单。

import { api } from '../api.js';
import { useEvents } from '../hooks/use-events.js';
import { FaultTimeline } from '../components/fault-timeline.js';
import { EventTable } from '../components/event-table.js';
import { escapeHtml } from '../components/elevator-card.js';
import { fmtTime } from '../constants.js';

export async function renderElevatorDetail(container, navigate, elevatorId) {
  const eventsHook = useEvents({ elevator_id: elevatorId });
  const [detail, faultsData, eventsData] = await Promise.all([
    api(`/api/elevators/${encodeURIComponent(elevatorId)}`),
    api(`/api/elevators/${encodeURIComponent(elevatorId)}/faults`),
    eventsHook.reload(),
  ]);

  const e = detail.elevator;
  const score = detail.score ? detail.score.score : e.health_score;
  const scoreClass = score <= 60 ? 'score-bad' : score <= 80 ? 'score-mid' : 'score-good';

  container.innerHTML = `
    <a href="/" data-internal-link class="back-link">← 返回总览</a>
    <h1 class="page-title">电梯详情：${escapeHtml(e.id)}</h1>
    <div class="detail-grid">
      <section class="panel">
        <h2 class="panel-title">台账信息</h2>
        <dl class="kv">
          <dt>楼栋</dt><dd>${escapeHtml(e.building)}</dd>
          <dt>型号</dt><dd>${escapeHtml(e.model)}</dd>
          <dt>安装日期</dt><dd>${escapeHtml(e.install_date || '-')}</dd>
          <dt>载重/楼层</dt><dd>${e.capacity_kg}kg / ${e.floors} 层</dd>
          <dt>当前状态</dt><dd>${escapeHtml(e.status_label || e.status || '-')}</dd>
          <dt>最近上报</dt><dd>${fmtTime(e.last_report_at)}</dd>
        </dl>
      </section>

      <section class="panel">
        <h2 class="panel-title">健康评分</h2>
        <div class="score-hero">
          <div class="score-circle ${scoreClass}">${score}</div>
          <div>${score.watchlisted ? '<span class="badge badge-danger">重点关注</span>' : '<span class="badge badge-success">正常</span>'}</div>
        </div>
        <ul class="score-reasons">
          ${(detail.score && detail.score.reasons || []).map((r) => `<li>${escapeHtml(r)}</li>`).join('')}
        </ul>
        <div class="muted">近30天故障 ${detail.score ? detail.score.fault_count : 0} 次 · 未按时处置 ${detail.score ? detail.score.untimely_count : 0} 次</div>
      </section>
    </div>

    <section class="panel">
      <h2 class="panel-title">模拟状态上报</h2>
      <form id="simulate-form" class="simulate-form">
        <input type="hidden" name="elevator_id" value="${escapeHtml(e.id)}" />
        <label>楼层 <input type="number" name="floor" value="5" min="0" max="${e.floors}" /></label>
        <label>位置 <input type="text" name="position" value="5F" /></label>
        <label>方向
          <select name="direction"><option value="idle">静止</option><option value="up">上行</option><option value="down">下行</option></select>
        </label>
        <label>门状态
          <select name="door"><option value="closed">关闭</option><option value="open">打开</option></select>
        </label>
        <label>平层 <select name="leveling"><option value="true">是</option><option value="false">否</option></select></label>
        <label>故障码 <input type="text" name="fault_code" placeholder="如 E01（留空无故障）" /></label>
        <label>乘客信号
          <select name="passenger_signal">
            <option value="none">无</option><option value="alarm">警铃</option>
            <option value="infrared">红外</option><option value="both">警铃+红外</option>
          </select>
        </label>
        <button type="submit" class="btn btn-primary">上报状态</button>
      </form>
      <div id="simulate-result" class="simulate-result"></div>
    </section>

    <section class="panel">
      <h2 class="panel-title">故障码时间线</h2>
      <div id="fault-timeline"></div>
    </section>

    <section class="panel">
      <h2 class="panel-title">该电梯困人事件</h2>
      <div id="elevator-events"></div>
    </section>
  `;

  container.querySelector('#fault-timeline').appendChild(FaultTimeline(faultsData.faults || []));
  const eventsBox = container.querySelector('#elevator-events');
  eventsBox.appendChild(EventTable(eventsData.events || [], {
    actions: [{ name: 'detail', label: '详情', kind: 'info', handler: (id) => navigate(`/events/${id}`) }],
  }));

  container.querySelector('#simulate-form').addEventListener('submit', async (ev) => {
    ev.preventDefault();
    const form = ev.target;
    const body = {
      floor: Number(form.floor.value),
      position: form.position.value,
      direction: form.direction.value,
      door: form.door.value,
      leveling: form.leveling.value === 'true',
      fault_code: form.fault_code.value.trim(),
      passenger_signal: form.passenger_signal.value,
      alarm_active: form.passenger_signal.value === 'alarm' || form.passenger_signal.value === 'both',
      infrared_active: form.passenger_signal.value === 'infrared' || form.passenger_signal.value === 'both',
    };
    const box = container.querySelector('#simulate-result');
    try {
      const res = await api(`/api/elevators/${encodeURIComponent(e.id)}/states`, { method: 'POST', body });
      let html = `<div class="ok">✅ 上报成功（状态：${res.entrapment_state}，累计 ${res.consecutive_seconds}s）</div>`;
      if (res.diagnosis) html += `<div class="warn">🧯 故障诊断：${escapeHtml(res.diagnosis.diagnosis)}</div>`;
      if (res.entrapment_event) html += `<div class="danger">🚨 触发困人事件：${escapeHtml(res.entrapment_event.id)}</div>`;
      box.innerHTML = html;
    } catch (err) {
      box.innerHTML = `<div class="err">❌ 上报失败：${escapeHtml(err.message)}</div>`;
    }
  });
}
