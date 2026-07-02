import { Link } from "react-router-dom";

const features = [
  {
    title: "Real-time collaboration",
    body: "Cards, edits, and votes sync instantly to everyone on the board over WebSockets.",
    icon: (
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="M4 12a8 8 0 0 1 13.66-5.66M20 12a8 8 0 0 1-13.66 5.66"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
        />
        <path
          d="M17 3v4h-4M7 21v-4h4"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    ),
  },
  {
    title: "Vote on what matters",
    body: "Surface the team's top priorities—participants vote and the important items rise to the top.",
    icon: (
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="m5 13 4 4L19 7"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    ),
  },
  {
    title: "Share with a link",
    body: "Create a board, send the link, and teammates join in seconds—no setup required.",
    icon: (
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="M10 13a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1.5 1.5M14 11a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1.5-1.5"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    ),
  },
  {
    title: "Lightweight & self-hostable",
    body: "A single Go binary with an embedded SQLite database. No external services to run.",
    icon: (
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <rect
          x="3"
          y="4"
          width="18"
          height="7"
          rx="1.5"
          stroke="currentColor"
          strokeWidth="2"
        />
        <rect
          x="3"
          y="13"
          width="18"
          height="7"
          rx="1.5"
          stroke="currentColor"
          strokeWidth="2"
        />
        <path
          d="M7 7.5h.01M7 16.5h.01"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
        />
      </svg>
    ),
  },
];

const steps = [
  {
    title: "Create a board",
    body: "Name your retro and organize feedback into columns like What went well and Action items.",
  },
  {
    title: "Share the link",
    body: "Invite the team with a single link—everyone joins the same live board.",
  },
  {
    title: "Vote in real time",
    body: "Add cards, cast votes, and watch priorities emerge together, instantly.",
  },
];

function BoardMockup() {
  return (
    <svg
      className="hero-mock"
      viewBox="0 0 460 300"
      role="img"
      aria-label="Preview of a VoteRun retrospective board with three columns of cards and votes"
    >
      <rect x="0" y="0" width="460" height="300" rx="14" fill="#ffffff" />
      <rect x="0" y="0" width="460" height="46" rx="14" fill="#f5f6f8" />
      <rect x="0" y="30" width="460" height="16" fill="#f5f6f8" />
      <circle cx="26" cy="23" r="6" fill="#3b71ca" />
      <rect x="42" y="18" width="96" height="10" rx="5" fill="#d7dde8" />
      <rect x="360" y="15" width="84" height="18" rx="9" fill="#e8eefb" />
      <circle cx="373" cy="24" r="4" fill="#14a44d" />
      <rect x="384" y="20" width="48" height="8" rx="4" fill="#9db4dd" />

      {/* Column 1 — green accent */}
      <g>
        <rect x="18" y="62" width="132" height="4" rx="2" fill="#14a44d" />
        <rect x="18" y="76" width="72" height="9" rx="4.5" fill="#c9cdd3" />
        <g>
          <rect x="18" y="96" width="132" height="52" rx="8" fill="#ffffff" stroke="#e0e0e0" />
          <rect x="30" y="108" width="96" height="7" rx="3.5" fill="#c9cdd3" />
          <rect x="30" y="121" width="72" height="7" rx="3.5" fill="#dfe2e6" />
          <rect x="30" y="134" width="42" height="12" rx="6" fill="#3b71ca" />
        </g>
        <g>
          <rect x="18" y="158" width="132" height="46" rx="8" fill="#ffffff" stroke="#e0e0e0" />
          <rect x="30" y="170" width="84" height="7" rx="3.5" fill="#c9cdd3" />
          <rect x="30" y="190" width="42" height="12" rx="6" fill="#eef1f5" />
        </g>
      </g>

      {/* Column 2 — red accent */}
      <g>
        <rect x="164" y="62" width="132" height="4" rx="2" fill="#dc4c64" />
        <rect x="164" y="76" width="64" height="9" rx="4.5" fill="#c9cdd3" />
        <g>
          <rect x="164" y="96" width="132" height="46" rx="8" fill="#ffffff" stroke="#e0e0e0" />
          <rect x="176" y="108" width="90" height="7" rx="3.5" fill="#c9cdd3" />
          <rect x="176" y="128" width="42" height="12" rx="6" fill="#3b71ca" />
        </g>
        <g>
          <rect x="164" y="152" width="132" height="52" rx="8" fill="#ffffff" stroke="#e0e0e0" />
          <rect x="176" y="164" width="96" height="7" rx="3.5" fill="#c9cdd3" />
          <rect x="176" y="177" width="60" height="7" rx="3.5" fill="#dfe2e6" />
          <rect x="176" y="190" width="42" height="12" rx="6" fill="#eef1f5" />
        </g>
      </g>

      {/* Column 3 — blue accent */}
      <g>
        <rect x="310" y="62" width="132" height="4" rx="2" fill="#54b4d3" />
        <rect x="310" y="76" width="80" height="9" rx="4.5" fill="#c9cdd3" />
        <g>
          <rect x="310" y="96" width="132" height="46" rx="8" fill="#ffffff" stroke="#e0e0e0" />
          <rect x="322" y="108" width="88" height="7" rx="3.5" fill="#c9cdd3" />
          <rect x="322" y="128" width="42" height="12" rx="6" fill="#3b71ca" />
        </g>
        <g>
          <rect x="310" y="158" width="132" height="46" rx="8" fill="#ffffff" stroke="#e0e0e0" />
          <rect x="322" y="170" width="72" height="7" rx="3.5" fill="#c9cdd3" />
          <rect x="322" y="190" width="42" height="12" rx="6" fill="#eef1f5" />
        </g>
      </g>
    </svg>
  );
}

export default function Landing() {
  return (
    <div className="landing">
      <section className="landing-hero">
        <div className="hero-copy">
          <span className="hero-eyebrow">Real-time retrospectives</span>
          <h1>Run better retros, together.</h1>
          <p className="hero-lead">
            VoteRun is a real-time retrospective app for agile teams. Create a
            board, collect feedback across columns, and vote on what matters—all
            updating live on every participant's screen.
          </p>
          <div className="hero-actions">
            <Link
              to="/login"
              state={{ mode: "register" }}
              className="btn btn-primary"
            >
              Get started
            </Link>
            <Link to="/login" className="btn btn-outline">
              Sign in
            </Link>
          </div>
          <p className="hero-note">Free to use · Self-hostable · No credit card</p>
        </div>
        <div className="hero-visual">
          <BoardMockup />
        </div>
      </section>

      <section className="landing-features">
        {features.map((f) => (
          <div className="feature" key={f.title}>
            <span className="feature-icon">{f.icon}</span>
            <h3>{f.title}</h3>
            <p>{f.body}</p>
          </div>
        ))}
      </section>

      <section className="landing-steps">
        <h2>How it works</h2>
        <ol>
          {steps.map((s, i) => (
            <li key={s.title}>
              <span className="step-num">{i + 1}</span>
              <div>
                <h3>{s.title}</h3>
                <p>{s.body}</p>
              </div>
            </li>
          ))}
        </ol>
      </section>

      <section className="landing-cta">
        <h2>Ready to run your next retro?</h2>
        <p>Spin up a board in seconds and invite your team.</p>
        <Link to="/login" state={{ mode: "register" }} className="btn btn-primary">
          Get started free
        </Link>
      </section>
    </div>
  );
}
