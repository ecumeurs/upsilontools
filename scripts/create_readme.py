

# explore all folders recursively. If no README.md file is present, create one.
# If a README.md file is present, check if it contains a link to the parent folder.
# If not, add it.


import os
import sys
import re

if len(sys.argv) < 2:
    print("Usage: %s <root>" % sys.argv[0])
    sys.exit(1)
root = sys.argv[1]
initRoot = root

def main(root):
    for r, dirs, files in os.walk(root):
        if ".git" in r:
            continue
        if "README.md" not in files:
            print("Creating README.md for %s" % r)
            with open(os.path.join(r, "README.md"), "w") as f:
                f.write("# %s" % r)
                f.write("\n\n[Up](../README.md)\n")
        else:
            print("Checking README.md for %s" % r)
            with open(os.path.join(r, "README.md"), "r") as f:
                content = f.read()
                if re.search(r"\[(.*)\]\((.*)\)", content) is None:
                    print("Adding link to parent folder for %s" % r)
                    with open(os.path.join(r, "README.md"), "a") as f:
                        f.write("\n\n[Up](../README.md)\n")
        # recurse into subfolders
        for d in dirs:
            if ".git" in d:
                continue
            main(os.path.join(r, d))

main(root)




