// useElevators(poll)：电梯列表轮询 hook。
// 被「电梯总览」与「重点关注」两个页面共用。
// poll=true 时每 5 秒自动刷新；返回 { elevators, loading, error, reload }。

import { api } from '../api.js';

const POLL_INTERVAL_MS = 5000;

export function useElevators(poll = false, query = '') {
  let elevators = [];
  let loading = true;
  let error = null;
  let timer = null;

  async function load() {
    try {
      const qs = query ? `?q=${encodeURIComponent(query)}` : '';
      const data = await api(`/api/elevators${qs}`);
      elevators = data.elevators || [];
      error = null;
    } catch (err) {
      error = err;
    } finally {
      loading = false;
    }
  }

  async function reload() {
    await load();
    return elevators;
  }

  function startPolling() {
    stopPolling();
    timer = setInterval(load, POLL_INTERVAL_MS);
  }

  function stopPolling() {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  }

  return {
    get elevators() { return elevators; },
    get loading() { return loading; },
    get error() { return error; },
    reload,
    startPolling,
    stopPolling,
  };
}
