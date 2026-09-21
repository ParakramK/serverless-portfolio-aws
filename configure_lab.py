#!/usr/bin/env python3
"""Local-only Learner Lab credential helper."""  # noqa: EXE001

import getpass
import os
import subprocess
import sys


def run(cmd):
    r = subprocess.run(cmd, check=False, capture_output=True, text=True)
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main():
    profile = input("Profile [lab]: ").strip() or "lab"
    region = input("Region [us-east-1]: ").strip() or "us-east-1"

    if os.environ.get("AWS_ENDPOINT_URL"):
        print(
            "WARN: AWS_ENDPOINT_URL is set (floci). Unset it in your shell after this:"
        )
        print("  unset AWS_ENDPOINT_URL")
        print()

    print(
        "Paste lab keys locally (hidden, not echoed). Get fresh values from Vocareum AWS Details."
    )
    key = getpass.getpass("AWS_ACCESS_KEY_ID: ").strip()
    secret = getpass.getpass("AWS_SECRET_ACCESS_KEY: ").strip()
    token = getpass.getpass("AWS_SESSION_TOKEN: ").strip()

    if not key or not secret or not token:
        print("ERROR: all three values required.", file=sys.stderr)
        sys.exit(1)
    if not key.startswith("ASIA"):
        print("WARN: key does not start with ASIA - ensure fresh Learner Lab keys.")

    for field, val in [
        ("aws_access_key_id", key),
        ("aws_secret_access_key", secret),
        ("aws_session_token", token),
        ("region", region),
    ]:
        code, out = run(["aws", "configure", "set", field, val, "--profile", profile])
        if code != 0:
            print(f"ERROR setting {field}: {out}", file=sys.stderr)
            sys.exit(1)

    print(f"Configured profile '{profile}'. Verifying (keys not shown)...")
    env = dict(os.environ)
    env.pop("AWS_ENDPOINT_URL", None)
    env["AWS_PROFILE"] = profile
    env["AWS_DEFAULT_REGION"] = region
    code, out = run(["aws", "sts", "get-caller-identity", "--region", region])
    # Re-run with env override via subprocess env
    if code != 0:
        # retry with explicit env to avoid floci leakage
        p = subprocess.run(
            ["aws", "sts", "get-caller-identity", "--region", region],
            check=False,
            capture_output=True,
            text=True,
            env=env,
        )
        code, out = p.returncode, (p.stdout or "") + (p.stderr or "")
    if code != 0:
        print("FAILED: " + out[-800:], file=sys.stderr)
        sys.exit(1)
    print("SUCCESS:")
    print(out[:800])
    print("\nNext:")
    print("  unset AWS_ENDPOINT_URL")
    print("  export AWS_PROFILE=lab")
    print("  sam build --template template.yaml")


if __name__ == "__main__":
    main()
