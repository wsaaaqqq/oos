#!/usr/bin/env python3
"""Sync GitHub Release assets to Gitee Release."""

import argparse
import json
import os
import subprocess
import sys
import tempfile
import urllib.request
import urllib.error


GITHUB_REPO = "wsaaaqqq/oos"
GITEE_REPO = "haitao666/oos"
GITEE_API = "https://gitee.com/api/v5"


def get_gitee_token():
    proc = subprocess.run(
        ["git", "credential", "fill"],
        input="url=https://gitee.com\n\n",
        capture_output=True, text=True,
    )
    for line in proc.stdout.splitlines():
        if line.startswith("password="):
            return line.split("=", 1)[1].strip()
    raise RuntimeError(
        "Gitee token not found. Run: git credential fill url=https://gitee.com\n"
        "Or set GITEE_TOKEN environment variable."
    )


def gitee_request(method, path, token, data=None, files=None):
    url = f"{GITEE_API}{path}"
    headers = {"Authorization": f"token {token}"}

    if files:
        import http.client
        import mimetypes
        boundary = "----FormBoundary" + os.urandom(16).hex()
        body = b""
        for key, val in data.items():
            body += f"--{boundary}\r\n".encode()
            body += f'Content-Disposition: form-data; name="{key}"\r\n\r\n'.encode()
            body += str(val).encode() + b"\r\n"
        for field_name, file_info in files.items():
            filename, file_content = file_info
            body += f"--{boundary}\r\n".encode()
            body += f'Content-Disposition: form-data; name="{field_name}"; filename="{filename}"\r\n'.encode()
            body += f"Content-Type: application/octet-stream\r\n\r\n".encode()
            body += file_content + b"\r\n"
        body += f"--{boundary}--\r\n".encode()
        headers["Content-Type"] = f"multipart/form-data; boundary={boundary}"
        req = urllib.request.Request(url, data=body, headers=headers, method=method)
    elif data:
        req = urllib.request.Request(
            url,
            data=json.dumps(data).encode(),
            headers={**headers, "Content-Type": "application/json"},
            method=method,
        )
    else:
        req = urllib.request.Request(url, headers=headers, method=method)

    try:
        resp = urllib.request.urlopen(req)
        return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        print(f"  API error {e.code}: {body[:300]}", file=sys.stderr)
        return None


def get_github_release(repo, tag=None):
    if tag:
        url = f"https://api.github.com/repos/{repo}/releases/tags/{tag}"
    else:
        url = f"https://api.github.com/repos/{repo}/releases/latest"
    with urllib.request.urlopen(url) as resp:
        return json.loads(resp.read())


def download_assets(release):
    files = {}
    tmpdir = tempfile.mkdtemp(prefix="oos_sync_")
    for asset in release.get("assets", []):
        name = asset["name"]
        url = asset["browser_download_url"]
        print(f"  Downloading {name} ...", end=" ", flush=True)
        local = os.path.join(tmpdir, name)
        urllib.request.urlretrieve(url, local)
        with open(local, "rb") as f:
            files[name] = f.read()
        print("OK")
    return files, tmpdir


def get_or_create_gitee_release(token, tag_name, name, body):
    # Check existing
    releases = gitee_request("GET", f"/repos/{GITEE_REPO}/releases", token)
    if releases:
        for r in releases:
            if r["tag_name"] == tag_name:
                print(f"  Gitee release already exists (id={r['id']})")
                return r

    # Create new
    print(f"  Creating Gitee release for {tag_name} ...")
    data = {
        "tag_name": tag_name,
        "name": name,
        "body": body,
        "target_commitish": "master",
        "prerelease": False,
    }
    result = gitee_request("POST", f"/repos/{GITEE_REPO}/releases", token, data=data)
    if result:
        print(f"  Created release id={result.get('id')}")
    return result


def upload_asset(token, release_id, filename, content):
    data = {"name": filename}
    files = {"file": (filename, content)}
    return gitee_request(
        "POST",
        f"/repos/{GITEE_REPO}/releases/{release_id}/attach_files",
        token,
        data=data,
        files=files,
    )


def sync_release(tag=None):
    print(f"=== Sync GitHub {GITHUB_REPO} -> Gitee {GITEE_REPO} ===\n")

    token = os.environ.get("GITEE_TOKEN") or get_gitee_token()
    print("Token acquired.\n")

    # Get GitHub release
    print(f"Fetching GitHub release (tag={tag or 'latest'}) ...")
    github_release = get_github_release(GITHUB_REPO, tag)
    tag_name = github_release["tag_name"]
    title = github_release.get("name", tag_name)
    body = github_release.get("body", "") or ""
    print(f"  Tag: {tag_name}  Title: {title}\n")

    # Download assets
    print("Downloading GitHub assets ...")
    files, tmpdir = download_assets(github_release)

    # Get or create Gitee release
    gitee_release = get_or_create_gitee_release(token, tag_name, title, body)
    if not gitee_release:
        print("ERROR: Could not create Gitee release", file=sys.stderr)
        sys.exit(1)

    # Get existing assets on Gitee
    release_id = gitee_release["id"]
    existing = gitee_request(
        "GET",
        f"/repos/{GITEE_REPO}/releases/{release_id}/attach_files",
        token,
    )
    existing_names = set()
    if existing:
        for a in existing:
            existing_names.add(a.get("name", ""))

    # Upload missing
    print(f"\nUploading to Gitee release ...")
    uploaded = 0
    for name, content in sorted(files.items()):
        if name in existing_names:
            print(f"  {name} - already exists, skipping")
            continue
        print(f"  Uploading {name} ({len(content)} bytes) ...", end=" ", flush=True)
        result = upload_asset(token, release_id, name, content)
        if result:
            print("OK")
            uploaded += 1
        else:
            print("FAILED")

    # Cleanup
    for name in files:
        os.remove(os.path.join(tmpdir, name))
    os.rmdir(tmpdir)

    print(f"\nDone. {uploaded} new assets uploaded.")
    print(f"View: https://gitee.com/{GITEE_REPO}/releases/{tag_name}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Sync GitHub Release assets to Gitee Release"
    )
    parser.add_argument(
        "--tag",
        help="Release tag (default: latest)",
        default=None,
    )
    args = parser.parse_args()
    sync_release(tag=args.tag)
