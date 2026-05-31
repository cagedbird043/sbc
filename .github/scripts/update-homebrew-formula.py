#!/usr/bin/env python3
"""Update the sbc homebrew formula version and sha256 checksums."""
import re
import sys

formula_path = sys.argv[1]
version = sys.argv[2]
sha_linux_amd64 = sys.argv[3]
sha_linux_arm64 = sys.argv[4]
sha_darwin_arm = sys.argv[5]
sha_completion = sys.argv[6] if len(sys.argv) > 6 else ""

with open(formula_path) as f:
    content = f.read()

# Update version
content = re.sub(r'version ".*"', f'version "{version}"', content)

# Update release URL version in url lines: match sbc-{platform} download URL
# The formula has url "https://github.com/cagedbird043/sbc/releases/download/v#{version}/sbc-{platform}"
# Resource completion URL uses literal version: url "https://.../download/v{version}/_sbc"
release_version_pattern = r'(releases/download/)v[^/]+(/sbc-)'
content = re.sub(release_version_pattern, rf'\g<1>v{version}\g<2>', content)

# Update sha256 lines (they follow url lines)
platforms = [
    ('sbc-Linux-amd64', sha_linux_amd64),
    ('sbc-Linux-arm64', sha_linux_arm64),
    ('sbc-Darwin-arm64', sha_darwin_arm),
]

for url_suffix, new_sha in platforms:
    # Find the url line containing the platform
    url_pattern = re.escape(f'{url_suffix}"')
    url_match = re.search(url_pattern, content)
    if url_match:
        after_url = content[url_match.end():]
        sha_line_match = re.search(r'sha256 "[^"]*"', after_url)
        if sha_line_match:
            old_sha = sha_line_match.group()
            new = f'sha256 "{new_sha}"'
            content = content.replace(old_sha, new, 1)

# Update completion resource version URL and SHA
if sha_completion:
    # Match the version in the completion resource URL (uses literal version, not interpolation)
    completion_url_pattern = r'(releases/download/)v[^/]+/_sbc"\)'
    content = re.sub(completion_url_pattern, rf'\g<1>v{version}/_sbc")', content)

    resource_match = re.search(r'resource "completion"', content)
    if resource_match:
        after_resource = content[resource_match.end():]
        sha_line_match = re.search(r'sha256 "[^"]*"', after_resource)
        if sha_line_match:
            old_sha = sha_line_match.group()
            content = content.replace(old_sha, f'sha256 "{sha_completion}"', 1)

with open(formula_path, 'w') as f:
    f.write(content)

print(f"Updated {formula_path} to version {version}")
