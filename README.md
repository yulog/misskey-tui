# Misskey-TUI

A TUI client for Misskey.

> [!NOTE]
> 大部分をGemini CLIで生成し、一部調整のみ人間が行っています。

## Features

- **Multiple Timelines**: Switch between Home, Local, Social, and Global timelines.
- **Post Details**: View detailed information about a post, including replies.
- **Create Posts**: Write and publish new posts.
- **Reply**: Reply to other users' posts.
- **Reactions**: React to posts with emojis. Custom emojis are consolidated into a single heart reaction.
- **Renotes**: Renote posts to share them with your followers.
- **Status Bar**: A status bar at the bottom of the screen displays your username and instance.
- **Word Wrapping**: Long posts are properly wrapped to fit the screen width.

## How to Use

1.  Create a `config.json` file with your instance URL and access token:
    ```json
    {
      "instance_url": "https://your.misskey.instance",
      "access_token": "YOUR_ACCESS_TOKEN"
    }
    ```
2.  Run the application:
    ```bash
    go run ./cmd/misskey-tui
    ```
    or using nix
    ```bash
    nix run
    ```

## Keybindings

- `h/l/s/g`: Switch between timelines (Home/Local/Social/Global).
- `p`: Create a new post.
- `enter`: View post details.
- `r`: React to the selected post (with ❤️).
- `R`: Reply to the selected post.
- `t`: Renote the selected post.
- `q`/`ctrl+c`: Quit the application.
