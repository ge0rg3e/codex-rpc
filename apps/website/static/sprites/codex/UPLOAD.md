# Discord artwork

For a public HTTPS website, the client uses the hosted PNG URLs automatically.

For local development, upload these files in your Discord application → Rich Presence → Art Assets. Use exactly these asset keys:

Extract the root `discord-assets.zip` and upload all ten PNGs inside it. Every image is 512 × 512, meeting Discord's minimum upload size. The logo is already named `codex-icon.png` so every filename matches its required key (without `.png`). Do not upload the source sprite sheet.

| File | Asset key |
| --- | --- |
| codex-idle.png | codex-idle |
| codex-thinking.png | codex-thinking |
| codex-coding.png | codex-coding |
| codex-editing.png | codex-editing |
| codex-running.png | codex-running |
| codex-waiting.png | codex-waiting |
| codex-success.png | codex-success |
| codex-error.png | codex-error |
| codex-stopped.png | codex-stopped |
| codex-icon.png (logo) | codex-icon |

The archive's `codex-icon.png` is the 512 × 512 version generated from the root `logo.png`, rather than the smaller web logo. You can also use it for the application's General Information → App Icon. Replace the old image if you previously uploaded a different logo.

Copy the application's Application ID into the tray app's settings, keep Discord desktop open, and enable Discord presence. Asset keys are used automatically with the localhost website setting.

See `manifest.json` for the source grid and chosen frames.
