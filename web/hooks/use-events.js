// useEvents(filter)：困人事件列表 hook。
// 被「困人事件」页与「电梯总览」页共用。
// filter: { status, elevator_id }；可通过 setFilter 动态更新。

import { api } from '../api.js';

export function useEvents(filter = {}) {
  let currentFilter = { ...filter };
  let events = [];
  let loading = true;
  let error = null;

  function buildQuery() {
    const params = new URLSearchParams();
    if (currentFilter.status) params.set('status', currentFilter.status);
    if (currentFilter.elevator_id) params.set('elevator_id', currentFilter.elevator_id);
    params.set('limit', '100');
    return params.toString();
  }

  async function load() {
    try {
      const data = await api(`/api/events?${buildQuery()}`);
      events = data.events || [];
      error = null;
    } catch (err) {
      error = err;
    } finally {
      loading = false;
    }
  }

  async function reload() {
    await load();
    return events;
  }

  function setFilter(next) {
    currentFilter = { ...currentFilter, ...next };
  }

  return {
    get events() { return events; },
    get loading() { return loading; },
    get error() { return error; },
    setFilter,
    reload,
  };
}
