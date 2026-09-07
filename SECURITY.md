# Security Policy

## Reporting a vulnerability

If you find a security issue (e.g. something in `config.json`/`state.json`
handling that could leak an API key, or a parsing bug that could be
triggered by a malicious/crafted SavedVariables or chat log file), please
**do not open a public issue**. Instead use GitHub's private reporting:

1. Go to the repo's **Security** tab → **Report a vulnerability**.
2. Describe the issue and, if possible, steps to reproduce.

You'll get an acknowledgment as soon as possible, and a fix or mitigation
will be prioritized before any public disclosure.

## Scope

This is a small personal/guild utility that runs locally and talks to one
API endpoint you configure yourself. There's no bug bounty, but responsible
disclosure is appreciated and credited in release notes if you'd like.

Note that `config.json` contains your API key in plain text — it's
git-ignored by default, but treat that file itself as sensitive on disk.
