module github.com/NetrisTV/ws-qvh

go 1.15

require (
	github.com/danielpaulus/go-ios v0.0.0-20191119131658-c495aaebbeb6
	github.com/danielpaulus/quicktime_video_hack v0.0.0-20200913112742-92dee353674c
	github.com/gorilla/websocket v1.4.1
	github.com/sirupsen/logrus v1.6.0
)

replace github.com/danielpaulus/quicktime_video_hack v0.0.0-20200913112742-92dee353674c => github.com/NetrisTV/quicktime_video_hack v0.0.0-20201026161452-fe5cb4b55736
