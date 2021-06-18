# Stash Proxy

Stash Proxy was created to solve an issue arising out of hosting Stash on a low-powered server - when live transcoding videos, the server is put under intense load. Stash proxy is a lightweight proxy to your actual Stash instance that transcodes video files locally instead of via the server.

It can optionally be configured to run a chrome instance with a temporary user directory in an incognito window and will clean up the user directory when exiting chrome.

This is a very early prototype, so bugs are likely to occur.
