# ws qvh

Web server for streaming the screen of iOS devices over WebSocket.

## How it works?

1. [danielpaulus/quicktime_video_hack](https://github.com/danielpaulus/quicktime_video_hack) - video streaming
2. [appium/WebDriverAgent](https://github.com/appium/WebDriverAgent) - device control
3. [NetrisTV/ws-scrcpy](https://github.com/NetrisTV/ws-scrcpy) - user interface
4. [NetrisTV/ws-qvh](https://github.com/NetrisTV/ws-qvh) - glues it all together

## Steps to set up

1. Get macOS (streaming only should also work on GNU/Linux)
2. Connect a device, accept "Trust This Computer".
3. Verify that you can record your device screen with QuickTime
4. Install [danielpaulus/quicktime_video_hack](https://github.com/danielpaulus/quicktime_video_hack) and verify that you can record your device screen with it
5. Build sources: `go build`. This command will produce `ws-qvh` binary.
6. Setup [appium/WebDriverAgent](https://github.com/appium/WebDriverAgent). `WebDriverAgent` directory must be placed near with `ws-qvh` binary. Places to look at:
   * [WebDriverAgent/README.md](https://github.com/appium/WebDriverAgent/blob/master/README.md)
   * [Appium XCUITest Driver Real Device Setup](http://appium.io/docs/en/drivers/ios-xcuitest-real-devices/)
   * [/wda-build.xconfig](/wda-build.xcconfig)
   * [/wdaStarter.go](/wdaStarter.go#L97)
7. Setup frontend:
   * Follow the instructions [here](https://github.com/NetrisTV/ws-scrcpy#ws-qvh)
   * Copy `dist` directory near to `ws-qvh` binary (or you can point full path to it in the second argument)
8. Run with command: `./ws-qvh [ADDRESS-TO-LISTEN-ON [PATH-TO-FRONTEND]]`:
   * Default address is `:8080`, which means all interfaces, TCP port 8080
   * Default path to frontend is `dist`
9. Open [http://127.0.0.1:8080/](http://127.0.0.1:8080/) in your browser.
   
# Notes

* Only video stream is transmitted (no audio).
* `WebDriverAgent` can be started only after the start on video transmission (i.e. quicktime interface activation).
* `WebDriverAgent` can take a long time to start.
* Control capabilities are very limited (compared to scrcpy/ws-scrcpy): 
   * single tap
   * `home` button click
   * swipe (this command will be sent only after the gesture is complete)
* No way to customize stream parameters (bitrate, fps, video size, etc.)
