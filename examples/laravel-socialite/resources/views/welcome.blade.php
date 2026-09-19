<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Sign in — Bohurupee</title>
    <link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 64 64'%3E%3Cpath fill='%23b45309' d='M12 43C11 35 13 31 15 33C16 34 17 34.5 18 35C23 33 28 33 32 34C37 30 40 22 38 13C37 9 35 6 33 4.5C43 9 51 19 52 31C52.5 34 52.5 36 52 38.5C52 46 49 52 44 55C39 58 32 58 26 56C19 54 13 51 12 45ZM16 44C20 40 26 41 29 46C24 49 18 48 16 44ZM48 44C44 40 38 41 35 46C40 49 46 48 48 44Z'/%3E%3C/svg%3E">
    <style>
        :root {
            --ink: #1c1917;
            --muted: #57534e;
            --line: #e7e5e4;
            --paper: #f6f1ea;
            --card: #fffdfb;
            --amber: #b45309;
        }

        * { box-sizing: border-box; }

        body {
            margin: 0;
            min-height: 100vh;
            color: var(--ink);
            background:
                radial-gradient(ellipse 36rem 20rem at 50% -6rem, #fde8c8 0%, transparent 70%),
                var(--paper);
            font-family: ui-sans-serif, system-ui, sans-serif;
            line-height: 1.5;
        }

        .screen {
            min-height: 100vh;
            display: grid;
            place-items: center;
            padding: 2.5rem 1.25rem;
        }

        .card {
            width: min(24rem, 100%);
            background: var(--card);
            border: 1px solid #e6dfd4;
            border-radius: 1.15rem;
            padding: 2rem 1.6rem 1.5rem;
            box-shadow: 0 18px 40px rgba(28, 25, 23, 0.06);
        }

        .mark { display: block; width: 3.25rem; height: auto; }

        h1 {
            margin: 1.1rem 0 0;
            font-size: 1.7rem;
            letter-spacing: -0.03em;
            line-height: 1.15;
        }

        .lede {
            margin: 0.4rem 0 0;
            color: var(--muted);
            font-size: 0.95rem;
        }

        .actions {
            display: grid;
            gap: 0.65rem;
            margin-top: 1.5rem;
        }

        .btn {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 0.7rem;
            min-height: 2.75rem;
            padding: 0.55rem 0.9rem;
            border-radius: 0.7rem;
            border: 1px solid var(--line);
            text-decoration: none;
            font-weight: 600;
            font-size: 0.95rem;
            color: var(--ink);
            background: #fff;
            transition: transform 0.12s ease, box-shadow 0.12s ease, border-color 0.12s ease;
        }

        .btn:hover {
            transform: translateY(-1px);
            box-shadow: 0 6px 16px rgba(28, 25, 23, 0.08);
        }

        .btn:focus-visible {
            outline: 2px solid var(--amber);
            outline-offset: 2px;
        }

        .btn svg { width: 1.15rem; height: 1.15rem; flex: none; }

        .github {
            color: #fff;
            background: #24292f;
            border-color: #24292f;
        }

        .note {
            margin: 1.25rem 0 0;
            padding-top: 1rem;
            border-top: 1px solid #efe8df;
            color: var(--muted);
            font-size: 0.8rem;
        }

        code {
            font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
            font-size: 0.92em;
        }
    </style>
</head>
<body>
    <main class="screen">
        <section class="card">
            <svg class="mark" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" fill="none" stroke="#b45309" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" role="img" aria-label="Bohurupee">
                <g transform="translate(3 0)">
                    <path d="M9 43C8 35 10 31 12 33C13 34 14 34.5 15 35C20 33 25 33 29 34C34 30 37 22 35 13C34 9 32 6 30 4.5C40 9 48 19 49 31C49.5 34 49.5 36 49 38.5C49 46 46 52 41 55C36 58 29 58 23 56C16 54 10 51 9 45Z"/>
                    <path d="M13 44C17 40 23 41 26 46C21 49 15 48 13 44Z"/>
                    <path d="M45 44C41 40 35 41 32 46C37 49 43 48 45 44Z"/>
                </g>
            </svg>
            <h1>Sign in</h1>
            <p class="lede">Continue with a local identity. These buttons never call Google or GitHub.</p>
            <div class="actions">
                <a class="btn" href="/login/google">
                    <svg viewBox="0 0 18 18" aria-hidden="true">
                        <path fill="#4285F4" d="M17.64 9.2c0-.64-.06-1.25-.16-1.84H9v3.48h4.84a4.14 4.14 0 0 1-1.8 2.72v2.26h2.92c1.7-1.57 2.68-3.88 2.68-6.62z"/>
                        <path fill="#34A853" d="M9 18c2.43 0 4.47-.8 5.96-2.18l-2.92-2.26c-.8.54-1.84.86-3.04.86-2.34 0-4.32-1.58-5.03-3.7H.96v2.33A9 9 0 0 0 9 18z"/>
                        <path fill="#FBBC05" d="M3.97 10.72A5.4 5.4 0 0 1 3.68 9c0-.6.1-1.18.29-1.72V4.95H.96A9 9 0 0 0 0 9c0 1.45.35 2.82.96 4.05l3.01-2.33z"/>
                        <path fill="#EA4335" d="M9 3.58c1.32 0 2.5.45 3.44 1.35l2.58-2.58C13.46.89 11.43 0 9 0A9 9 0 0 0 .96 4.95l3.01 2.33C4.68 5.16 6.66 3.58 9 3.58z"/>
                    </svg>
                    Continue with Google
                </a>
                <a class="btn github" href="/login/github">
                    <svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                        <path d="M8 0C3.58 0 0 3.58 0 8a8 8 0 0 0 5.47 7.59c.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82A7.7 7.7 0 0 1 8 3.87c.68 0 1.36.09 2 .26 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8 8 0 0 0 16 8c0-4.42-3.58-8-8-8z"/>
                    </svg>
                    Continue with GitHub
                </a>
            </div>
            <p class="note">Bohurupee should be listening on <code>127.0.0.1:4190</code>.</p>
        </section>
    </main>
</body>
</html>
