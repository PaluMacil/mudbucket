# Mud Bucket

This is a quick, naive way to access files with a throwaway credential instead of logging into your sensitive fileshare or leaving the files entirely unprotected or trusting the public machine with a USB stick. Mud Bucket serves files from a static directory and provides user authentication via a token.

**Do not leave this running!** There is no protection again brute force token-guessing and once your token is used, you should consider that your token and maybe even your files are leaked since public computers should always be treated as entirely and comprehensively compromised.

## Features

- Secure and non-secure server modes: Secure only means that the server will create a self-signed certificate. This isn't meant for longterm use or sharing with others. You'll need to click through the browser warning because of the self-signed certificate. Insecure is for quicker development and debugging.
- User authentication using a token. Selection of a long token that's hard to guess is on the user. **Once you access the files from the public computer, shut down the server and pick a new token before you run this again.**
- List of static files in the static directory. You'll need to place these files manually as there is no upload functionality.
- Serving static files: The list above are hyperlinks.
- Logout: this removes the token cookie, which is a good idea to prevent snooping from the next user, though you should consider the token immediately compromised the moment you typed it into a public computer.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/)
- [systemd](https://www.freedesktop.org/wiki/Software/systemd/) ...if you want to run the application as a service. This is likely part of your Linux distribution. You can execute this app on other operating systems as well, but I have not documented service management.

### Installation

1. Clone the repository:

    ```bash
    git clone https://github.com/PaluMacil/mudbucket.git
    cd mudbucket
    ```

2. Build the application:

    ```bash
    go build
    ```

This will create an executable file in your current directory. The template files are embedded in the binary, so the binary includes them and is portable without them.

### Manual Usage

Run the application with the following command (add `--secure` to server it over TLS):

```bash
./mudbucket
```

You can set several environmental variables:

- **PORT**: the port on which the server will run (default is "8483").
- **STATIC_DIR**: the directory from which static files will be served (default is "./static").
- **TOKEN_VALUE**: the token for user authentication (default is "token123" which you should absolutely not use).
- **CERT_DIR**: the directory where the certificates for the secure server are stored (default is "./certs").

### Running as a Service

You can run the application as a service using systemd (on Linux) to manage restart after reboots and to set env vars. An example systemd service file is provided in the repository. To use it, copy it to the appropriate location and **replace the placeholder env vars**:

```bash
sudo cp mudbucket.service /etc/systemd/system/mudbucket.service
sudo nano /etc/systemd/system/mudbucket.service

sudo systemctl start app_name.service
sudo systemctl enable app_name.service
```

## Who should use this

Anybody desperate to use a public computer connected to a free printer. This is at least better than sticking a USB drive into it or logging into a Google Drive.

Do not use this as a standing server. I did not mitigate enough likely security risks to recommend this for a normal production or even personal persistent use.

## License

This project is licensed under the MIT License - see the LICENSE.md file for details.
