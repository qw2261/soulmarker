// G6 完成门槛「5 倍预测峰值压测 30 分钟」可复现剧本。
//
// 用法（在真实 staging 上，k6 需单独安装）：
//   # 1) 基线：探测当前容量（低速率、短时长），记录观测到的吞吐与延迟
//   k6 run --env BASE_URL=https://<staging-domain> \
//          --env TARGET_RPS=5 --env DURATION=1m --env MODE=baseline scripts/loadtest.js
//
//   # 2) 峰值：按预测峰值 RPS 的 5 倍量级持续 30 分钟（真实 staging，独立库/独立域）
//   k6 run --env BASE_URL=https://<staging-domain> \
//          --env TARGET_RPS=<5x_peak_rps> --env DURATION=30m --env MODE=peak \
//          --env EVENT_ID=<id> scripts/loadtest.js
//
// 判定（对齐 goal.md 初始 SLO / G6 完成门槛）：
//   - error_rate < 0.01（关键 API 5xx < 1%，Public Beta 目标）
//   - read P95 < 500 ms（普通读取 P95 < 500 ms）
//   - checkin P95 < 800 ms
//   - 全程无 Critical/High、无 unhandled 500；写路径 register 使用独立 staging 库
//
// 注意：默认只打**公开读路径**；`ENABLE_WRITE=1` 时叠加真实报名写路径（需真实活动
// 与票种，且压测后应清理或使用隔离数据库）。预测峰值 RPS 由基线实测 + 业务峰值
// 预估确定，本脚本把 TARGET_RPS 作为可注入目标，不对数值硬编码。

import http from 'k6/http';
import { check } from 'k6';
import { Rate, Counter } from 'k6/metrics';

const BASE = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
const EVENT_ID = __ENV.EVENT_ID || '1';
const MODE = __ENV.MODE || 'peak';
const TARGET_RPS = Number(__ENV.TARGET_RPS || 20);
const DURATION = __ENV.DURATION || '30m';
const ENABLE_WRITE = __ENV.ENABLE_WRITE === '1';
const READ_RATIO = Number(__ENV.READ_RATIO || 0.9); // 读占比，写占比 = 1 - READ_RATIO

export const errorRate = new Rate('error_rate');
export const checkins = new Counter('checkins');

export const options = {
  scenarios: {
    [MODE]: {
      executor: 'constant-arrival-rate',
      rate: TARGET_RPS,       // 每秒到达数（RPS）
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 100,
      maxVUs: 1000,
    },
  },
  thresholds: {
    error_rate: ['rate<0.01'],        // 关键 API 5xx < 1%（Public Beta SLO）
    http_req_duration: ['p(95)<500'], // 普通读取 P95 < 500 ms（读混写时以读为准）
  },
};

function mark(res, label) {
  const ok = res.status >= 200 && res.status < 500 && res.status !== 429 && res.status !== 503;
  errorRate.add(!ok);
  check(res, { [`${label} http ${res.status}`]: () => true });
}

export default function () {
  // 读路径（公开匿名端点）
  mark(http.get(`${BASE}/api/v1/events`), 'listEvents');
  mark(http.get(`${BASE}/api/v1/events/${EVENT_ID}`), 'getEvent');
  mark(http.get(`${BASE}/api/v1/events/${EVENT_ID}/tickets`), 'listTickets');
  checkins.add(0);

  // 写路径（可选）：真实报名，需活动与票种存在，且压测需使用隔离库
  if (ENABLE_WRITE && (Math.random() > READ_RATIO)) {
    const body = JSON.stringify({
      // 字段与 RegisterEventRequest 对齐；真实返回值不入断言，仅追踪错误率
      name: 'loadtest-' + __VU + '-' + (__ITER % 1000),
      email: 'loadtest-' + __VU + '-' + (__ITER % 1000) + '@example.com',
    });
    mark(
      http.post(`${BASE}/api/v1/events/${EVENT_ID}/register`, body, {
        headers: { 'Content-Type': 'application/json' },
      }),
      'register'
    );
  }

  // 500ms 内略作间歇避免纯热点；constant-arrival-rate 已稳定 RPS，这里可省
}
