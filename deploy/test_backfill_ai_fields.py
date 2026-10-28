import json
import pathlib
import sqlite3
import tempfile
import unittest

from backfill_ai_fields import apply, extract


class BackfillExtractionTests(unittest.TestCase):
    def test_extract_preserves_ai_fields_before_cutoff(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            export = root / "heartbeats.json"
            stage = root / "stage.db"
            export.write_text(json.dumps({"days": [{"heartbeats": [
                {"id": "old", "time": 1.0, "ai_line_changes": -4,
                 "human_line_changes": 3, "ai_input_tokens": 10,
                 "ai_cached_input_tokens": 20, "ai_output_tokens": 5,
                 "ai_prompt_length": 7, "ai_session": "session"},
                {"id": "plain", "time": 1.1},
                {"id": "new", "time": 3.0, "ai_input_tokens": 99},
            ]}]}))
            result = extract(export, stage, 2000)
            self.assertEqual(result["staged"], 1)
            with sqlite3.connect(stage) as db:
                row = db.execute("select origin_id,ai_line_changes,human_line_changes,ai_input_tokens,ai_cached_input_tokens,ai_output_tokens,a_iprompt_length,ai_session from ai_patch").fetchone()
            self.assertEqual(row, ("old", -4, 3, 10, 20, 5, 7, "session"))

            live = root / "live.db"
            backup = root / "backup.db"
            with sqlite3.connect(live) as db:
                db.execute("create table heartbeats (origin text, origin_id text, ai_line_changes integer, "
                           "human_line_changes integer, ai_input_tokens integer, ai_cached_input_tokens integer, "
                           "ai_output_tokens integer, a_iprompt_length integer, ai_session text)")
                db.execute("insert into heartbeats(origin,origin_id) values ('wakatime','old')")
            self.assertEqual(apply(live, stage), {"staged": 1, "matched": 1})
            outcome = apply(live, stage, backup, True)
            self.assertEqual(outcome["updated"], 1)
            self.assertEqual(outcome["integrity"], "ok")
            self.assertTrue(backup.exists())
            with sqlite3.connect(live) as db:
                values = db.execute("select ai_input_tokens,ai_cached_input_tokens from heartbeats").fetchone()
            self.assertEqual(values, (10, 20))


if __name__ == "__main__":
    unittest.main()
