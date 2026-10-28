#!/usr/bin/env python3
"""Backfill AI fields from a WakaTime heartbeat export into existing Wakapi rows.

Run `extract` on a machine with enough memory for the JSON export. Transfer the
small staging SQLite file to the Wakapi host, inspect `apply` without --apply,
then run `apply --apply --backup ...` on the host.
"""

import argparse
import json
import pathlib
import sqlite3

FIELDS = ("ai_line_changes", "human_line_changes", "ai_input_tokens",
          "ai_cached_input_tokens", "ai_output_tokens", "a_iprompt_length", "ai_session")


def extract(export_path, stage_path, cutoff_ms):
    export_path, stage_path = pathlib.Path(export_path), pathlib.Path(stage_path)
    if stage_path.exists():
        raise FileExistsError(stage_path)
    with export_path.open("rb") as source:
        data = json.load(source)
    eligible = 0
    with sqlite3.connect(stage_path) as db:
        db.execute("create table ai_patch (origin_id text primary key, ai_line_changes integer not null, "
                   "human_line_changes integer not null, ai_input_tokens integer not null, "
                   "ai_cached_input_tokens integer not null, ai_output_tokens integer not null, "
                   "a_iprompt_length integer not null, ai_session text not null)")
        pending = []
        for day in data.get("days", []):
            for heartbeat in day.get("heartbeats", []):
                origin_id = heartbeat.get("id")
                if not origin_id or int(heartbeat.get("time", 0) * 1000) >= cutoff_ms:
                    continue
                values = (int(heartbeat.get("ai_line_changes") or 0),
                          int(heartbeat.get("human_line_changes") or 0),
                          int(heartbeat.get("ai_input_tokens") or 0),
                          int(heartbeat.get("ai_cached_input_tokens") or 0),
                          int(heartbeat.get("ai_output_tokens") or 0),
                          int(heartbeat.get("ai_prompt_length") or 0),
                          str(heartbeat.get("ai_session") or ""))
                if not any(values):
                    continue
                eligible += 1
                pending.append((origin_id, *values))
                if len(pending) >= 1000:
                    db.executemany("insert or ignore into ai_patch values (?,?,?,?,?,?,?,?)", pending)
                    pending.clear()
        if pending:
            db.executemany("insert or ignore into ai_patch values (?,?,?,?,?,?,?,?)", pending)
        staged = db.execute("select count(*) from ai_patch").fetchone()[0]
        integrity = db.execute("pragma integrity_check").fetchone()[0]
        if integrity != "ok":
            raise RuntimeError(f"staging database integrity: {integrity}")
    stage_path.chmod(0o600)
    return {"eligible": eligible, "staged": staged, "bytes": stage_path.stat().st_size}


def apply(live_path, stage_path, backup_path=None, do_apply=False):
    live_path, stage_path = pathlib.Path(live_path), pathlib.Path(stage_path)
    with sqlite3.connect(live_path, timeout=30) as db:
        db.execute("pragma busy_timeout=30000")
        columns = {row[1] for row in db.execute("pragma table_info(heartbeats)")}
        if not set(FIELDS).issubset(columns):
            raise RuntimeError("Wakapi AI schema migration has not run")
        db.execute("attach database ? as stage", (str(stage_path),))
        staged = db.execute("select count(*) from stage.ai_patch").fetchone()[0]
        matched = db.execute("select count(*) from heartbeats h join stage.ai_patch p "
                             "on h.origin_id=p.origin_id where h.origin='wakatime'").fetchone()[0]
        result = {"staged": staged, "matched": matched}
        if not do_apply:
            return result
        if not backup_path:
            raise ValueError("--backup is required with --apply")
        if pathlib.Path(backup_path).exists():
            raise FileExistsError(backup_path)
        with sqlite3.connect(backup_path) as backup:
            db.backup(backup)
            if backup.execute("pragma integrity_check").fetchone()[0] != "ok":
                raise RuntimeError("backup integrity check failed")
        pathlib.Path(backup_path).chmod(0o600)
        with db:
            db.execute("create index if not exists idx_ai_backfill_origin on heartbeats(origin_id)")
            db.execute("update heartbeats set "
                       "ai_line_changes=p.ai_line_changes, human_line_changes=p.human_line_changes, "
                       "ai_input_tokens=p.ai_input_tokens, ai_cached_input_tokens=p.ai_cached_input_tokens, "
                       "ai_output_tokens=p.ai_output_tokens, a_iprompt_length=p.a_iprompt_length, "
                       "ai_session=p.ai_session from stage.ai_patch p "
                       "where heartbeats.origin='wakatime' and heartbeats.origin_id=p.origin_id")
            result["updated"] = db.execute("select changes()").fetchone()[0]
            db.execute("drop index idx_ai_backfill_origin")
        result["integrity"] = db.execute("pragma integrity_check").fetchone()[0]
        if result["integrity"] != "ok":
            raise RuntimeError("post-backfill integrity check failed")
        return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    subcommands = parser.add_subparsers(dest="command", required=True)
    extract_parser = subcommands.add_parser("extract")
    extract_parser.add_argument("--export", required=True)
    extract_parser.add_argument("--stage", required=True)
    extract_parser.add_argument("--cutoff-ms", required=True, type=int)
    apply_parser = subcommands.add_parser("apply")
    apply_parser.add_argument("--live", required=True)
    apply_parser.add_argument("--stage", required=True)
    apply_parser.add_argument("--backup")
    apply_parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    if args.command == "extract":
        print(extract(args.export, args.stage, args.cutoff_ms))
    else:
        print(apply(args.live, args.stage, args.backup, args.apply))
