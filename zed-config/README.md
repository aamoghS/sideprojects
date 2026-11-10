# zed-config

My Zed setup on Windows.

- `settings.json` goes in `%APPDATA%\Zed\settings.json`. It registers Claude Agent and Cursor as external agents (they use the existing Claude Code and Cursor CLI logins, not API keys), installs the LaTeX extension, and makes texlab build with `latexmk` on save and jump to the PDF afterwards.
- `SumatraPDF-settings.txt` goes in `%LOCALAPPDATA%\SumatraPDF\`. It makes double-clicking in the PDF jump back to that line in Zed.

Discord presence comes from [`../zed-discord-presence`](../zed-discord-presence/).
