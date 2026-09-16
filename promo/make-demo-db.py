#!/usr/bin/env python3
"""Generate a synthetic opencode.db for recording the oos demo GIF.

No real user data: 6 fake sessions across 3 fake projects.
Timestamps are relative to *now* so the TUI always shows
today/yesterday dates no matter when you re-record.

Usage:
    python promo/make-demo-db.py [output.db]
Default output:
    %TEMP%/oosdemo/.local/share/opencode/opencode.db
"""
import json
import os
import sqlite3
import sys
import tempfile
import time

HOUR = 3600 * 1000

# (session_id, title, slug, directory, age_ms, first_user_msg)
SESSIONS = [
    ("ses_demo01aaaa", "login bug", "login-bug",
     "/home/demo/projects/my-api", 2 * 3600 * 1000,
     "Help me fix the login page bug"),
    ("ses_demo02bbbb", "race condition", "race-condition",
     "/home/demo/projects/payment-service", 3 * 3600 * 1000,
     "Fix race condition in order flow, root cause was a subtle bug"),
    ("ses_demo03cccc", "bundle size", "bundle-size",
     "/home/demo/projects/frontend-app", 22 * 3600 * 1000,
     "Dashboard bundle bug, apply the hotfix before release"),
    ("ses_demo04dddd", "jwt auth", "jwt-auth",
     "/home/demo/projects/my-api", 2 * 24 * 3600 * 1000,
     "Add user authentication with JWT"),
    ("ses_demo05eeee", "getting started", "getting-started",
     "/home/demo/projects/docs-site", 3 * 24 * 3600 * 1000,
     "Rewrite the getting-started guide"),
    ("ses_demo06ffff", "backup cron", "backup-cron",
     "/home/demo/projects/infra-scripts", 4 * 24 * 3600 * 1000,
     "Nightly backup cron keeps failing"),
]

ASSISTANT_REPLY = "Sure, let me look into it."


def main() -> None:
    if len(sys.argv) > 1:
        db_path = sys.argv[1]
    else:
        db_path = os.path.join(
            tempfile.gettempdir(), "oosdemo", ".local",
            "share", "opencode", "opencode.db",
        )
    os.makedirs(os.path.dirname(db_path), exist_ok=True)
    if os.path.exists(db_path):
        os.remove(db_path)

    now = int(time.time() * 1000)
    db = sqlite3.connect(db_path)
    cur = db.cursor()
    cur.execute("""CREATE TABLE session (
        id TEXT PRIMARY KEY, title TEXT, slug TEXT, directory TEXT,
        model TEXT, agent TEXT, parent_id TEXT,
        time_updated INTEGER, time_archived INTEGER)""")
    cur.execute("""CREATE TABLE message (
        id TEXT PRIMARY KEY, session_id TEXT, data TEXT,
        time_created INTEGER)""")
    cur.execute("""CREATE TABLE part (
        message_id TEXT, data TEXT, time_created INTEGER)""")

    # compact separators to match Go's json.Marshal (oos queries
    # look for '"type":"text"' with no spaces)
    def js(obj):
        return json.dumps(obj, separators=(",", ":"))

    model = js({"id": "claude-opus-4-6", "providerID": "anthropic"})
    for i, (sid, title, slug, directory, age, text) in enumerate(SESSIONS):
        ts = now - age
        cur.execute(
            "INSERT INTO session VALUES (?,?,?,?,?,?,?,?,?)",
            (sid, title, slug, directory, model, "build", None, ts, None),
        )
        umsg = f"msg_demo{i:02d}u"
        amsg = f"msg_demo{i:02d}a"
        cur.execute(
            "INSERT INTO message VALUES (?,?,?,?)",
            (umsg, sid, js({"role": "user"}), ts),
        )
        cur.execute(
            "INSERT INTO message VALUES (?,?,?,?)",
            (amsg, sid, js({"role": "assistant"}), ts + 1000),
        )
        cur.execute(
            "INSERT INTO part VALUES (?,?,?)",
            (umsg, js({"type": "text", "text": text}), ts),
        )
        cur.execute(
            "INSERT INTO part VALUES (?,?,?)",
            (amsg, js({"type": "text", "text": ASSISTANT_REPLY}),
             ts + 1000),
        )

    db.commit()

    # sanity check with the same shapes oos queries
    n_session = cur.execute(
        "SELECT COUNT(*) FROM session WHERE time_archived IS NULL "
        "AND (parent_id IS NULL OR parent_id = '')").fetchone()[0]
    n_text = cur.execute(
        "SELECT COUNT(*) FROM part WHERE data LIKE '%\"type\":\"text\"%'"
    ).fetchone()[0]
    db.close()
    print(f"wrote {db_path}")
    print(f"sessions={n_session} text_parts={n_text}")


if __name__ == "__main__":
    main()
