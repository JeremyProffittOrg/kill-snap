import json
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


class ZipLifecycleTest(unittest.TestCase):
    def test_writer_and_deployer_use_existing_zip_policy(self):
        rules = json.loads((ROOT / "lifecycle-policy.json").read_text())["Rules"]
        rule = next(r for r in rules if r["ID"] == "DeleteProjectZipsAfter30Days")
        prefix = rule["Filter"]["Prefix"]
        self.assertEqual(prefix, "project-zips/")
        self.assertEqual(rule["Expiration"]["Days"], 30)
        self.assertFalse("projects/a/original.cr3".startswith(prefix))
        self.assertIn('"project-zips/%s/', (ROOT / "lambda/zip/raw_index.go").read_text())
        workflow = (ROOT / ".github/workflows/deploy.yml").read_text()
        self.assertIn("file://lifecycle-policy.json", workflow)
        self.assertNotIn("cat > /tmp/lifecycle.json", workflow)
        template = (ROOT / "template.yaml").read_text()
        self.assertNotIn("AWS::CloudWatch::Alarm", template)
        self.assertNotIn("AWS::CloudWatch::Dashboard", template)
        for package in ("zip", "api"):
            makefile = (ROOT / "lambda" / package / "Makefile").read_text()
            self.assertIn("/bootstrap .", makefile)
            self.assertNotIn("/bootstrap main.go", makefile)


if __name__ == "__main__":
    unittest.main()
