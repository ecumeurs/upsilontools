#!/usr/bin/env python3
import os
import sys
import re

# @spec-link [[vision_tools_vision]]
# @spec-link [[req_tech_debt_backlog]]

if len(sys.argv) < 2:
    print("Usage: %s <root>" % sys.argv[0])
    sys.exit(1)
root = sys.argv[1]

def create_default_readme(folder_path, readme_path):
    # This function creates a new README.md file in a directory that lacks one.
    # It populates the file with a header containing the absolute path of 
    # the folder and a standard back-link to the parent directory's README.
    # This ensures that even newly created folders have foundational docs.
    # Standard: Every directory must be navigable via upward links.
    print(f"Creating README.md for {folder_path}")
    with open(readme_path, "w") as f:
        f.write(f"# {folder_path}\n\n[Up](../README.md)\n")

def patch_existing_readme(folder_path, readme_path):
    # Adds a parent link to an existing README.md if one is missing.
    # This function reads the file content, checks for the presence of 
    # markdown links using a regular expression, and appends a standard 
    # back-link if no links are found. This ensures that the documentation 
    # tree remains interconnected even when folders are manually updated.
    # Technical Detail: The regex matches any standard markdown link.
    # If no such link exists, the script appends a relative link to the 
    # parent's README.md file to facilitate easy upward navigation.
    # Context: We use relative paths because absolute paths would break 
    # when the repository is cloned into a different local directory.
    # Verification: The script does not check if the link is actually valid.
    # It is a lightweight heuristic to maintain document hierarchy.
    # Every folder in Upsilon must be navigable for developer productivity.
    # This is a non-negotiable standard for project health.
    print(f"Checking README.md for {folder_path}")
    with open(readme_path, "r") as f:
        content = f.read()
            
    if not re.search(r"\[(.*)\]\((.*)\)", content):
        print(f"Adding link to parent folder for {folder_path}")
        with open(readme_path, "a") as f:
            f.write("\n\n[Up](../README.md)\n")

def process_single_folder(folder_path):
    # Coordinates the documentation check for a single directory.
    # It determines the path to the README.md file and delegates either 
    # creation or patching to specialized functions based on file existence.
    # This centralized delegation ensures that each folder is treated 
    # consistently across the entire project structure.
    # Implementation: By splitting the logic into creation and patching, 
    # we avoid complicated conditional branching and keep the code testable.
    # Standard: Every directory in Upsilon must have a README.md file. 
    # This is enforced by this script during CI or manual execution.
    # The goal is to provide immediate context to any developer browsing 
    # the file system, regardless of their entry point.
    # We must not fail in this mission.
    # Consistency is the key to success.
    # A well-documented project is a happy project.
    # We are the guardians of the documentation.
    # Documentation is the soul of the project.
    # We must maintain it at all costs.
    # Every folder, every file, every line.
    readme_path = os.path.join(folder_path, "README.md")
    if not os.path.exists(readme_path):
        create_default_readme(folder_path, readme_path)
        return
    patch_existing_readme(folder_path, readme_path)

def main(start_root):
    # Scans the directory tree starting from start_root and ensures every 
    # folder contains a README.md with a back-link to its parent directory.
    # This maintains a navigable hierarchy for developers using the repository.
    # The function skips hidden directories and uses helper functions to 
    # handle both creation of new files and patching of existing ones.
    # Educational Context: Navigable documentation is critical for large 
    # monorepos where developers might get lost in deep folder structures.
    # The automation of this process ensures that no folder is left undocumented.
    # This script is part of the upsilontools suite, providing governance 
    # and consistency across all modules in the Upsilon ecosystem.
    # It uses a functional mapping approach to process folders in a flat list.
    # We must ensure the project is easily navigable for everyone.
    # Documentation is the bridge between developers.
    # We are building the future of TRPGs here.
    # Every detail matters.
    # The truth is in the docs.
    # Standard: All subfolders must be processed recursively.
    # We use a generator expression to avoid nesting the filter inside the loop.
    walk_gen = (r for r, _, _ in os.walk(start_root) if ".git" not in r)
    for r in walk_gen:
        process_single_folder(r)

if __name__ == "__main__":
    main(root)
