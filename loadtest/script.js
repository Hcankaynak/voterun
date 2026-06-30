import http from "k6/http";
import { check } from "k6";
import { Trend, Counter } from "k6/metrics";
import { WebSocket } from "k6/websockets";

// ---------------------------------------------------------------------------
// VoteRun load test
//
// Scenario: one facilitator (the only registered account) creates a single
// shared board in setup(). 150 anonymous virtual users then "open the link"
// (GET board + WebSocket), submit their answer cards, and a subset upvotes.
// Board participation APIs are unauthenticated, so the VUs never register.
// ---------------------------------------------------------------------------

// --- Config (override with -e KEY=value) -----------------------------------
const BASE_URL = (__ENV.BASE_URL || "https://api.voterun.app").replace(/\/+$/, "");
const WS_URL = (__ENV.WS_URL || BASE_URL.replace(/^http/, "ws")).replace(/\/+$/, "");
const VUS = Number(__ENV.VUS || 150);
const SMOKE = __ENV.SMOKE === "1" || __ENV.SMOKE === "true";

const CARDS_PER_USER = Number(__ENV.CARDS_PER_USER || 3);
const VOTE_RATIO = Number(__ENV.VOTE_RATIO || 0.55);
const SEED_CARDS = Number(__ENV.SEED_CARDS || 3);
const CLEANUP = __ENV.CLEANUP !== "0" && __ENV.CLEANUP !== "false";

const SESSION_DURATION_MS = Number(__ENV.SESSION || (SMOKE ? 30 : 120)) * 1000;
const ACTION_INTERVAL_MS = Number(__ENV.ACTION_INTERVAL || 4000);
const ACTION_JITTER_MS = Number(__ENV.ACTION_JITTER || 2000);

const JSON_HEADERS = { "Content-Type": "application/json" };

// --- Custom metrics --------------------------------------------------------
const wsConnectTime = new Trend("ws_connect_time", true);
const wsPropagation = new Trend("ws_propagation_ms", true);
const wsMessages = new Counter("ws_messages_received");
const wsSessions = new Counter("ws_sessions");
const wsErrors = new Counter("ws_errors");

// Sample retrospective answers used as card text.
const ANSWERS = [
  "Great team collaboration this sprint",
  "Deployments were smooth and fast",
  "Too many meetings interrupted focus time",
  "We shipped the feature ahead of schedule",
  "Flaky tests slowed down the pipeline",
  "Onboarding docs need updating",
  "Pairing sessions were really helpful",
  "Scope creep on the main epic",
  "Customer feedback loop improved a lot",
  "Need clearer ownership of tickets",
  "Good incident response handling",
  "Code reviews took too long",
];

function pick(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomId() {
  return Math.random().toString(36).slice(2, 10);
}

// Flatten all card IDs from a board object (board.columns[].cards[]).
function cardIdsFromBoard(board) {
  const ids = [];
  if (board && Array.isArray(board.columns)) {
    for (const col of board.columns) {
      if (col && Array.isArray(col.cards)) {
        for (const card of col.cards) {
          if (card && card.id) ids.push(card.id);
        }
      }
    }
  }
  return ids;
}

// --- Load profile ----------------------------------------------------------
export const options = {
  setupTimeout: "60s",
  scenarios: SMOKE
    ? {
        smoke: {
          executor: "ramping-vus",
          startVUs: 0,
          stages: [
            { duration: "15s", target: 5 },
            { duration: "45s", target: 5 },
            { duration: "10s", target: 0 },
          ],
          gracefulStop: "10s",
        },
      }
    : {
        main: {
          executor: "ramping-vus",
          startVUs: 0,
          stages: [
            { duration: "1m", target: VUS },
            { duration: "5m", target: VUS },
            { duration: "30s", target: 0 },
          ],
          gracefulStop: "15s",
        },
      },
  thresholds: {
    http_req_failed: ["rate<0.02"],
    http_req_duration: ["p(95)<800", "p(99)<2000"],
    checks: ["rate>0.98"],
    ws_connect_time: ["p(95)<2000"],
  },
};

// --- Setup: facilitator creates the single shared board --------------------
export function setup() {
  const email = `loadtest+${Date.now()}@voterun.test`;
  const password = "loadtest-pass-123";

  let res = http.post(
    `${BASE_URL}/api/auth/register`,
    JSON.stringify({ email, password, name: "Load Test Facilitator" }),
    { headers: JSON_HEADERS }
  );
  let token = res.status === 201 ? res.json("token") : null;

  if (!token && res.status === 409) {
    res = http.post(
      `${BASE_URL}/api/auth/login`,
      JSON.stringify({ email, password }),
      { headers: JSON_HEADERS }
    );
    token = res.status === 200 ? res.json("token") : null;
  }
  if (!token) {
    throw new Error(`facilitator auth failed: ${res.status} ${res.body}`);
  }

  const authHeaders = Object.assign({}, JSON_HEADERS, {
    Authorization: `Bearer ${token}`,
  });

  const created = http.post(
    `${BASE_URL}/api/boards`,
    JSON.stringify({ name: `Load Test ${new Date().toISOString()}` }),
    { headers: authHeaders }
  );
  if (created.status !== 201) {
    throw new Error(`create board failed: ${created.status} ${created.body}`);
  }
  const board = created.json();
  const boardId = board.id;
  const columnIds = (board.columns || []).map((c) => c.id);
  if (!boardId || columnIds.length === 0) {
    throw new Error(`board missing id/columns: ${created.body}`);
  }

  for (let i = 0; i < SEED_CARDS; i++) {
    const columnId = columnIds[i % columnIds.length];
    http.post(
      `${BASE_URL}/api/boards/${boardId}/cards`,
      JSON.stringify({ columnId, text: `Seed answer #${i + 1}`, author: "Facilitator" }),
      { headers: JSON_HEADERS }
    );
  }

  const snapshot = http.get(`${BASE_URL}/api/boards/${boardId}`);
  const cardIds = snapshot.status === 200 ? cardIdsFromBoard(snapshot.json()) : [];

  console.log(
    `[setup] board=${boardId} columns=${columnIds.length} seedCards=${cardIds.length} target=${BASE_URL}`
  );
  return { boardId, columnIds, cardIds };
}

// --- VU: one anonymous participant session ---------------------------------
export default function (data) {
  return new Promise((resolve) => {
    const voterId = `lt-${__VU}-${randomId()}`;
    const author = `User-${__VU}`;
    const isVoter = Math.random() < VOTE_RATIO;

    // "Open the link": initial board load over REST.
    const opened = http.get(`${BASE_URL}/api/boards/${data.boardId}`, {
      tags: { action: "open" },
    });
    check(opened, { "open board 200": (r) => r.status === 200 }, { action: "open" });

    let cardIds =
      opened.status === 200 ? cardIdsFromBoard(opened.json()) : data.cardIds.slice();
    let submitted = 0;
    let pendingAt = 0;
    let actionTimer = null;
    let closeTimer = null;
    let resolved = false;

    const connectStart = Date.now();
    const ws = new WebSocket(`${WS_URL}/ws/boards/${data.boardId}`);

    const finish = () => {
      if (actionTimer !== null) clearInterval(actionTimer);
      if (closeTimer !== null) clearTimeout(closeTimer);
      try {
        ws.close();
      } catch (e) {
        // ignore
      }
      if (!resolved) {
        resolved = true;
        resolve();
      }
    };

    const doAction = () => {
      if (submitted < CARDS_PER_USER) {
        // Submit an answer card.
        const columnId = data.columnIds[Math.floor(Math.random() * data.columnIds.length)];
        const text = `${pick(ANSWERS)} (${author} #${submitted + 1})`;
        const res = http.post(
          `${BASE_URL}/api/boards/${data.boardId}/cards`,
          JSON.stringify({ columnId, text, author }),
          { headers: JSON_HEADERS, tags: { action: "submit" } }
        );
        check(res, { "submit 201": (r) => r.status === 201 }, { action: "submit" });
        submitted++;
        pendingAt = Date.now();
      } else if (isVoter && cardIds.length > 0) {
        // Upvote a random card from the live board.
        const cardId = cardIds[Math.floor(Math.random() * cardIds.length)];
        const res = http.post(
          `${BASE_URL}/api/cards/${cardId}/vote`,
          JSON.stringify({ voterId }),
          { headers: JSON_HEADERS, tags: { action: "vote" } }
        );
        check(res, { "vote 200": (r) => r.status === 200 }, { action: "vote" });
        pendingAt = Date.now();
      }
      // Otherwise the user is idle, just listening to live updates.
    };

    ws.onopen = () => {
      wsConnectTime.add(Date.now() - connectStart);
      wsSessions.add(1);
      const interval = ACTION_INTERVAL_MS + Math.floor(Math.random() * ACTION_JITTER_MS);
      actionTimer = setInterval(doAction, interval);
    };

    ws.onmessage = (e) => {
      wsMessages.add(1);
      if (pendingAt) {
        wsPropagation.add(Date.now() - pendingAt);
        pendingAt = 0;
      }
      try {
        const msg = JSON.parse(e.data);
        if (msg && msg.board) {
          const ids = cardIdsFromBoard(msg.board);
          if (ids.length > 0) cardIds = ids;
        }
      } catch (err) {
        // ignore non-JSON frames
      }
    };

    ws.onerror = () => {
      wsErrors.add(1);
    };

    ws.onclose = () => {
      finish();
    };

    // End the session after a fixed duration, then the VU reconnects.
    closeTimer = setTimeout(finish, SESSION_DURATION_MS);
  });
}

// --- Teardown: best-effort cleanup of test cards ---------------------------
export function teardown(data) {
  if (!CLEANUP) return;
  const snapshot = http.get(`${BASE_URL}/api/boards/${data.boardId}`);
  if (snapshot.status !== 200) return;
  const ids = cardIdsFromBoard(snapshot.json());
  for (const id of ids) {
    http.del(`${BASE_URL}/api/cards/${id}`);
  }
  console.log(`[teardown] deleted ${ids.length} cards (board ${data.boardId} cannot be deleted via API)`);
}

// Compact, dependency-free console summary plus a full JSON report.
function fmt(n) {
  if (n === undefined || n === null || Number.isNaN(n)) return "-";
  return Math.round(n * 100) / 100;
}

function line(label, metric) {
  if (!metric) return `  ${label}: (no samples)`;
  const v = metric.values;
  if (v.rate !== undefined && v.passes !== undefined) {
    return `  ${label}: rate=${fmt(v.rate * 100)}% (pass ${v.passes}/${v.passes + v.fails})`;
  }
  if (v.rate !== undefined && v.count === undefined) {
    return `  ${label}: rate=${fmt(v.rate * 100)}%`;
  }
  if (v["p(95)"] !== undefined) {
    return `  ${label}: avg=${fmt(v.avg)} p95=${fmt(v["p(95)"])} p99=${fmt(v["p(99)"])} max=${fmt(v.max)}`;
  }
  if (v.count !== undefined) {
    return `  ${label}: count=${v.count}`;
  }
  return `  ${label}: ${JSON.stringify(v)}`;
}

export function handleSummary(data) {
  const m = data.metrics;
  const out = [
    "",
    "=== VoteRun load test summary ===",
    line("checks", m.checks),
    line("http_req_failed", m.http_req_failed),
    line("http_req_duration", m.http_req_duration),
    line("ws_connect_time", m.ws_connect_time),
    line("ws_propagation_ms", m.ws_propagation_ms),
    line("ws_sessions", m.ws_sessions),
    line("ws_messages_received", m.ws_messages_received),
    line("ws_errors", m.ws_errors),
    "",
    "Full report: loadtest/main-summary.json",
    "",
  ].join("\n");

  return {
    "loadtest/main-summary.json": JSON.stringify(data, null, 2),
    stdout: out,
  };
}
