import unittest

from plan import plan

class TestExamplePlan(unittest.TestCase):
    def test_no_changes(self):
        changed_files = []
        expected = {"examples": []}
        self.assertEqual(plan(changed_files), expected)

    def test_change_markdown_only(self):
        changed_files = ["README.md", "docs/overview.md"]
        expected = {"examples": []}
        self.assertEqual(plan(changed_files), expected)

    def test_change_single_example(self):
        changed_files = ["examples/webserver/action.yml"]
        expected = {"examples": ["webserver"]}
        self.assertEqual(plan(changed_files), expected)

    def test_change_multiple_examples(self):
        changed_files = [
            "examples/webserver/action.yml",
            "examples/sokol/action.yaml"
        ]
        expected = {"examples": ["sokol", "webserver"]}
        self.assertEqual(plan(changed_files), expected)

    def test_change_non_example_file(self):
        changed_files = ["some_other_file.py"]
        # Assuming all examples are ['webserver', 'sokol']
        expected = {"examples": ["sokol", "webserver"]}
        self.assertEqual(plan(changed_files), expected)

    def test_change_example_and_non_example_file(self):
        changed_files = [
            "examples/webserver/action.yml",
            "some_other_file.py"
        ]
        # Assuming all examples are ['webserver', 'sokol']
        expected = {"examples": ["sokol", "webserver"]}
        self.assertEqual(plan(changed_files), expected)

if __name__ == "__main__":
    unittest.main()
