#!/usr/bin/env python3

import json
import sys
from pathlib import Path

# The path for current script.
SCRIPT_PATH = Path(__file__).parent.absolute()
# The path for `.github` dir.
GITHUB_DIR = SCRIPT_PATH.parent.parent
# The project dir.
PROJECT_DIR = GITHUB_DIR.parent

def get_examples() -> list[str]:
    examples_dir = Path(f"{GITHUB_DIR}/examples")
    examples = []
    
    if not examples_dir.exists():
        return examples
    
    for item in examples_dir.iterdir():
        if item.is_dir():
            action_yml = item / "action.yml"
            action_yaml = item / "action.yaml"
            
            if action_yml.exists() or action_yaml.exists():
                examples.append(item.name)
    
    return sorted(examples)

def calculate_final_examples(changed_files: list[str], all_examples: list[str]) -> list[str]:
    # Ignore markdown files
    changed_files = [f for f in changed_files if not f.endswith(".md")]
    
    changed_examples = set()
    
    for file_path in changed_files:
        if file_path.startswith("examples/"):
            # run only the changed example
            path_parts = file_path.split("/")
            if len(path_parts) >= 2:
                example_name = path_parts[1]
                if example_name in all_examples:
                    changed_examples.add(example_name)
        else:
            # If a non-example file is changed, run all examples
            return all_examples
    
    
    return sorted(list(changed_examples))

def plan(changed_files: list[str]) -> dict:
    all_examples = get_examples()
    
    final_examples = calculate_final_examples(changed_files, all_examples)
    
    result = {
        "examples": final_examples
    }
    
    return result

if __name__ == "__main__":
    changed_files = sys.argv[1:]
    result = plan(changed_files)
    print(json.dumps(result))
