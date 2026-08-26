// 前后端共享枚举/常量定义（与 backend/domain/enums.go 保持一致）。
// 修改时需同时修改 Go 侧定义，README 中列有完整对照表。

export const EVENT_STATUS = {
  alerted: { label: '已告警', color: 'danger' },
  accepted: { label: '已接单', color: 'warn' },
  processing: { label: '处置中', color: 'info' },
  released: { label: '已解除', color: 'success' },
  escalated: { label: '已升级', color: 'danger' },
};

export const DOOR_STATUS = {
  open: '开门',
  closed: '关门',
};

export const DIRECTION = {
  up: '上行',
  down: '下行',
  idle: '静止',
};

export const FAULT_TYPE = {
  known: '已知故障',
  unknown: '未知故障',
};

export const PASSENGER_SIGNAL = {
  none: '无乘客信号',
  alarm: '警铃触发',
  infrared: '红外探测',
  both: '警铃+红外',
};

export const SEVERITY = {
  high: '高',
  medium: '中',
  low: '低',
};

export function eventStatusLabel(s) {
  return (EVENT_STATUS[s] && EVENT_STATUS[s].label) || s;
}

export function eventStatusColor(s) {
  return (EVENT_STATUS[s] && EVENT_STATUS[s].color) || 'info';
}

export function fmtTime(iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
