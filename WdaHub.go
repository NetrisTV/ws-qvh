package main

type WdaHub struct {
	stopSignal chan interface{}
	udid string
	wdaUrls map[string]*string
}

func NewWdaHub(stopSignal chan interface{}, udid string) *WdaHub {
	return &WdaHub{
		stopSignal: stopSignal,
		udid: udid,
		wdaUrls: make(map[string]*string),
	}
}

func (w WdaHub) getWdaUrl() string{
	var udid = w.udid
	if w.wdaUrls[udid] != nil {
		return *w.wdaUrls[udid]
	}
	ch := make(chan []byte)
	wdaProcess := NewWdaProcess(ch)
	var str = ""

	go func() {
		wdaProcess.Start(udid)
	}()
	result := <- ch
	if result != nil {
		str = string(result)
		w.wdaUrls[udid] = &str
	}
	return str
}