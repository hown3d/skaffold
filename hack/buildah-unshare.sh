#!/bin/sh
buildah unshare $HOME/go/bin/dlv dap --headless --listen=:2345