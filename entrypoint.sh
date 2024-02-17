#!/bin/bash

if [ "$BROWSER_HEADLESS" = "false" ]; then
    export DISPLAY=:1
    WINDOW_RESOLUTION=${WINDOW_RESOLUTION:-1280x720}

    # Start virtual display
    Xvfb $DISPLAY -screen 0 ${WINDOW_RESOLUTION}x16 &

    # Start x11vnc server
    x11vnc -display $DISPLAY -rfbport 5900 -forever -bg -nopw -quiet

    # Start the novnc proxy
    /opt/novnc/utils/novnc_proxy --vnc localhost:5900 --listen 6080 &
fi

# Start the program
./app
