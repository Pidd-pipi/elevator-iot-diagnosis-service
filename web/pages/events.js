// 困人事件页（/events）。
// 数据来源：GET /api/events；处置动作：accept / resolve / escalate。

import { api } from '../api.js';
import { useEvents } from '../hooks/use-events.js';
import { EventTable } from '../components/event-table.js';
import { eventStatusLabel } from '../constants.js';
import { escapeHtml } from '../components/elevator-card.js';
import { toast } from './overview.js';

export async function renderEvents(container, navigate) {
  const hook = useEvents();
  await hook.reload();

  const statusOptions = ['', 'alerted', 'accepted', 'processing', 'released', 'escalated']
    .map((s) => `<option value="${s}">${s === '' ? '全部状态' : eventStatusLabel(s)}</option>`).join('');

  container.innerHTML = `
    <h1 class="page-title">困人事件</h1>
    <div class="toolbar">
      <label>状态过滤
        <select id="event-filter" class="input">${statusOptions}</select>
      </label>
      <button id="event-refresh" class="btn btn-outline">刷新</button>
    </div>
    <div id="events-table"></div>
  `;

  const tableBox = container.querySelector('#events-table');
  function paint() {
    tableBox.innerHTML = '';
    tableBox.appendChild(EventTable(hook.events, {
      actions: [
        { name: 'accept', label: '接单', kind: 'primary', handler: (id) => doAccept(id) },
        { name: 'resolve', label: '处置完成', kind: 'success', handler: (id) => openResolveModal(id) },
        { name: 'escalate', label: '升级', kind: 'danger', handler: (id) => doEscalate(id) },
      ],
    }));
  }
  paint();

  container.querySelector('#event-filter').addEventListener('change', async (e) => {
    hook.setFilter({ status: e.target.value });
    await hook.reload();
    paint();
  });
  container.querySelector('#event-refresh').addEventListener('click', async () => {
    await hook.reload();
    paint();
    toast('已刷新');
  });

  async function doAccept(id) {
    try {
      await api(`/api/events/${id}/accept`, { method: 'POST' });
      toast('接单成功');
      await hook.reload();
      paint();
    } catch (err) {
      toast('接单失败：' + err.message, true);
    }
  }

  async function doEscalate(id) {
    if (!confirm('确认升级该事件并发送二次告警？')) return;
    try {
      await api(`/api/events/${id}/escalate`, { method: 'POST', body: { reason: '页面人工确认升级' } });
      toast('已升级并二次告警');
      await hook.reload();
      paint();
    } catch (err) {
      toast('升级失败：' + err.message, true);
    }
  }

  function openResolveModal(id) {
    const modal = document.createElement('div');
    modal.className = 'modal-mask';
    modal.innerHTML = `
      <div class="modal">
        <h3>处置完成：${escapeHtml(id)}</h3>
        <label>处置人 <input id="m-disposer" class="input" placeholder="必填" /></label>
        <label>处理措施 <textarea id="m-measure" class="input" placeholder="必填"></textarea></label>
        <label>备注 <input id="m-note" class="input" /></label>
        <label>恢复时间 <input id="m-recovery" class="input" type="datetime-local" value="${new Date().toISOString().slice(0, 16)}" /></label>
        <div class="modal-actions">
          <button id="m-cancel" class="btn btn-outline">取消</button>
          <button id="m-submit" class="btn btn-success">确认关闭</button>
        </div>
      </div>`;
    document.body.appendChild(modal);
    modal.querySelector('#m-cancel').addEventListener('click', () => modal.remove());
    modal.querySelector('#m-submit').addEventListener('click', async () => {
      const disposer = modal.querySelector('#m-disposer').value.trim();
      const measure = modal.querySelector('#m-measure').value.trim();
      const recovery = modal.querySelector('#m-recovery').value;
      try {
        await api(`/api/events/${id}/resolve`, {
          method: 'POST',
          body: {
            disposer,
            measure,
            recovery_time: new Date(recovery).toISOString(),
          },
        });
        toast('处置完成，事件已解除');
        modal.remove();
        await hook.reload();
        paint();
      } catch (err) {
        toast('处置失败：' + err.message, true);
      }
    });
  }
}
