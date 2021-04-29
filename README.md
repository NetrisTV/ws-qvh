# ws qvh

Web Socket server for streaming the screen of iOS devices.

## How it works?

1. [danielpaulus/quicktime_video_hack](https://github.com/danielpaulus/quicktime_video_hack) - video streaming
2. [appium/WebDriverAgent](https://github.com/appium/WebDriverAgent) - device control
3. [NetrisTV/ws-scrcpy](https://github.com/NetrisTV/ws-scrcpy) - user interface
4. [NetrisTV/ws-qvh](https://github.com/NetrisTV/ws-qvh) - forwards the video stream over Web Socket

## Steps to set up

1. Get macOS (streaming only should also work on GNU/Linux)
2. Connect a device, accept "Trust This Computer".
3. Verify that you can record your device screen with QuickTime
4. Install [danielpaulus/quicktime_video_hack](https://github.com/danielpaulus/quicktime_video_hack) and verify that you can record your device screen with it
5. Build sources: `go build`. This command will produce `ws-qvh` binary.
6. Make sure your `ws-qvh` binary is available via the `PATH` environment variable.
7. Setup and run `ws-scrcpy`. Follow the instructions [here](https://github.com/NetrisTV/ws-scrcpy#ws-qvh).
8. Open link provided by `ws-scrcpy` in your browser.
   
# Notes

* Only video stream is transmitted (no audio).
* `WebDriverAgent` can be started only after the start of video transmission (i.e. quicktime interface activation).
* `WebDriverAgent` can take a long time to start.
* Control capabilities are very limited (compared to scrcpy/ws-scrcpy): 
   * single tap
   * `home` button click
   * swipe (this command will be sent only after the gesture is complete)
* No way to customize stream parameters (bitrate, fps, video size, etc.)
