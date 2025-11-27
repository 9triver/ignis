// todo: 目前还有个bug,如果两方同时dial对方，可能会丢失一个datachannel，如果存在较远的先后关系，不会有这个问题
// todo: 资源回收过程可能不规范，有多余报错
// remained: 断点续接可能存在短时间不应期（state 变为 disconnected前）
// todo: waitGroup会不会比time.After好，在等待关闭的时候

package stun

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/9triver/ignis/transport/stun/common"
	eh "github.com/9triver/ignis/transport/stun/utils/errorhandler"
	sm "github.com/9triver/ignis/transport/stun/utils/safemap"
	"github.com/gorilla/websocket"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v3"
)

// var _ Sender = (*NatSender)(nil)``

var roomID = "default"

// 注意，由于包含了writeMu, 只能以指针传递，不能以值传递，因为sync.Mutex 是一个值类型，但它不应该被复制
type connWithMu struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *connWithMu) writeJSON(v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.conn.WriteJSON(v)
	eh.ErrorHandler(err, 2, eh.LogAndPanic)
}

func (c *connWithMu) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.conn.Close()
	eh.ErrorHandler(err, 2, eh.LogOnly)
}

type NatSender struct {
	Dst2dc *sm.SafeMap[string, *webrtc.DataChannel]
	Dst2pc *sm.SafeMap[string, *webrtc.PeerConnection]
	// 消息分派通道
	Dst2SdpCh *sm.SafeMap[string, chan []byte]
	// listening通道
	listeningSdpCh chan []byte
	// current Listening pc
	listeningPc     *webrtc.PeerConnection
	listeningCancel context.CancelFunc
	// accept同步机制
	canAccept chan struct{}
	// 用锁来保护多协程写，用chan确保单协程读
	signalingServerConn *connWithMu
	// 监听id
	source string

	Dst2ConnectionType *sm.SafeMap[string, string]
	// 用于同步和资源释放
	Ctx    context.Context
	cancel context.CancelFunc
	// 用于细零度资源释放
	dst2cancel *sm.SafeMap[string, context.CancelFunc]
	// 消息接收时的回调函数
	onMessage func(msg []byte)
	// used to init webrtc api
	settingEngine webrtc.SettingEngine
	webrtcConfig  webrtc.Configuration
	listeningFunc func()
}

func NewNatSender(source string, signalingServer string, onMessage func(msg []byte), logLevel logging.LogLevel) *NatSender {
	ctx, cancel := context.WithCancel(context.Background())
	ns := &NatSender{
		Dst2dc:             sm.NewSafeMap[string, *webrtc.DataChannel](),
		Dst2pc:             sm.NewSafeMap[string, *webrtc.PeerConnection](),
		Dst2SdpCh:          sm.NewSafeMap[string, chan []byte](),
		Dst2ConnectionType: sm.NewSafeMap[string, string](),
		listeningSdpCh:     make(chan []byte),
		listeningPc:        nil,
		canAccept:          make(chan struct{}),
		signalingServerConn: &connWithMu{
			conn: connectSignalingServer(signalingServer, roomID),
		},
		source:        source,
		Ctx:           ctx,
		dst2cancel:    sm.NewSafeMap[string, context.CancelFunc](),
		cancel:        cancel,
		onMessage:     onMessage,
		settingEngine: newSettingEngine(logLevel),
		webrtcConfig: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs:       []string{"turn:8.153.200.135:3479"},
					Username:   "test1",
					Credential: "123456",
				},
			},
		},
	}
	sendRegister(ns.signalingServerConn.conn, source)
	go ns.sdpDispatcher(ctx)
	return ns
}

var _ Sender = &NatSender{}

func (natSender *NatSender) sdpDispatcher(ctx context.Context) {
	for {
		// 当natSender关闭的时候，应当关闭conn,这样ReadMessage就不会阻塞
		// 正常情况下就是从此处退出
		_, rawMsg, err := natSender.signalingServerConn.conn.ReadMessage()
		if eh.ErrorHandler(err, 1, eh.LogOnly) {
			return
		}
		var msg common.SignalMsg
		var targetedMsg common.TargetedMsg
		err = json.Unmarshal(rawMsg, &msg)
		if eh.ErrorHandler(err, 1, eh.LogOnly) {
			continue
		}
		err = json.Unmarshal([]byte(msg.Data), &targetedMsg)
		if eh.ErrorHandler(err, 1, eh.LogOnly) {
			continue
		}

		dst := targetedMsg.Src
		ch, ok := natSender.Dst2SdpCh.Get(dst)
		if !ok {
			ch = natSender.listeningSdpCh
		}
		select {
		case <-ctx.Done():
			// 通过ctx退出
			// 此处若阻塞说明sdpHandler先退出
			return
		case ch <- rawMsg:
		}
	}
}

func connectSignalingServer(pSignalingServer string, rID string) *websocket.Conn {
	url := fmt.Sprintf("%s?room=%s", pSignalingServer, rID)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	eh.ErrorHandler(err, 1, eh.LogAndPanic)

	log.Println("connected to signaling server")
	return conn
}

func newSettingEngine(l logging.LogLevel) webrtc.SettingEngine {
	s := webrtc.SettingEngine{}

	loggerFactory := logging.NewDefaultLoggerFactory()
	loggerFactory.DefaultLogLevel = l
	s.LoggerFactory = loggerFactory

	s.SetICETimeouts(5*time.Second, 5*time.Second, 5*time.Second)
	return s
}

func newPeerConnection(s webrtc.SettingEngine, cfg webrtc.Configuration) *webrtc.PeerConnection {
	api := webrtc.NewAPI(webrtc.WithSettingEngine(s))
	peerConnection, err := api.NewPeerConnection(cfg)
	eh.ErrorHandler(err, 1, eh.LogAndFatal)
	return peerConnection
}

func newDataChannelOnOpen(dc *webrtc.DataChannel, pc *webrtc.PeerConnection, natSender *NatSender, dst *string, succeeded chan bool) func() {
	return func() {
		natSender.Dst2dc.Set(*dst, dc)
		natSender.Dst2pc.Set(*dst, pc)

		log.Printf("dc to %s is Open\n" + *dst)
		if succeeded != nil {
			succeeded <- true
		}
	}
}

func newPeerConnectionOnICEConnectionStateChange() func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		log.Printf("OnICEConnectionStateChange: %s\n", connectionState.String())
	}
}

func newPeerConnectionOnConnectionStateChange(p *webrtc.PeerConnection, natSender *NatSender, dst *string) func(s webrtc.PeerConnectionState) {
	return func(s webrtc.PeerConnectionState) {
		log.Printf("OnConnectionStateChange: %s\n", s.String())
		// 该部分应该作为网络连接波动时或连接无法建立时的资源回收
		// 在natSender操控peerConnection.close的接口的时候，不需要加上webrtc.PeerConnectionStateClosed
		// 因为在此情况下pc.close被调用，说明ns.close被调用，资源已经清理
		if s == webrtc.PeerConnectionStateDisconnected || s == webrtc.PeerConnectionStateFailed {
			cancel, ok := natSender.dst2cancel.Get(*dst)
			if !ok {
				// 防止资源重复释放
				log.Println("cancel is called more than once")
				return
			}
			cancel()
			time.Sleep(100 * time.Millisecond)
			pc, _ := natSender.Dst2pc.Get(*dst)
			dc, _ := natSender.Dst2dc.Get(*dst)
			ch, _ := natSender.Dst2SdpCh.Get(*dst)
			eh.ErrorHandler(dc.Close(), 1, eh.LogOnly)
			eh.ErrorHandler(pc.Close(), 1, eh.LogOnly)
			close(ch)
			natSender.dst2cancel.Del(*dst)
			natSender.Dst2dc.Del(*dst)
			natSender.Dst2SdpCh.Del(*dst)
			natSender.Dst2pc.Del(*dst)
			natSender.Dst2ConnectionType.Del(*dst)
		}
		if s == webrtc.PeerConnectionStateConnected {
			stats := p.GetStats()
			var candidateId string
			for _, report := range stats {
				r, ok := report.(webrtc.ICECandidatePairStats)
				if ok && r.Type == webrtc.StatsTypeCandidatePair && r.Nominated {
					candidateId = r.LocalCandidateID
				}
			}
			for _, report := range stats {
				r, ok := report.(webrtc.ICECandidateStats)
				if ok && r.ID == candidateId {
					t := r.CandidateType.String()
					ip := r.IP
					port := r.Port
					natSender.Dst2ConnectionType.Set(*dst, t)
					log.Printf("connectionType %s, address %s:%d \n", t, ip, port)
				}
			}
		}
	}
}

func newPeerConnectionOnSignalingStateChange() func(s webrtc.SignalingState) {
	return func(s webrtc.SignalingState) {
		log.Printf("OnSignalingStateChange: %s\n", s.String())
	}
}

func newPeerConnectionOnDataChannel(natSender *NatSender, dst *string, f func(msg []byte)) func(*webrtc.DataChannel) {
	return func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			natSender.Dst2dc.Set(*dst, dc)
			natSender.Dst2pc.Set(*dst, natSender.listeningPc)
			natSender.Dst2SdpCh.Set(*dst, make(chan []byte))
			natSender.dst2cancel.Set(*dst, natSender.listeningCancel)
			natSender.canAccept <- struct{}{}
			err := dc.SendText("hello from server")
			eh.ErrorHandler(err, 1, eh.LogAndPanic)
			log.Printf("dc to %s is open, id is %d\n", *dst, dc.ID())
		})

		dc.OnClose(func() {
			// 应该不会单独处理这个，外部不应该手动调用dc的close函数，也不应该看到dc的抽象，应该有peerConnection
			log.Println("listening dc closed with state " + dc.ReadyState().String())
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("OnMessage '%s':", *dst)
			f(msg.Data)
		})
	}
}

func newPeerConnectionOnICECandidate(c *connWithMu, src *string, dst *string) func(i *webrtc.ICECandidate) {
	return func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		eh.ErrorHandler(err, 1, eh.LogAndPanic)
		targetedMsg := common.TargetedMsg{
			Src:  *src,
			Dst:  *dst,
			Data: string(candidateString),
		}
		targetedString, err := json.Marshal(targetedMsg)
		eh.ErrorHandler(err, 1, eh.LogAndPanic)
		c.writeJSON(&common.SignalMsg{
			Type: common.Candidate,
			Data: string(targetedString),
		})
	}
}

// 该函数用于创建goroutine后没有退出，是为了可能存在的连接建立后交换消息的情况，目前不清楚需不需要在连接建立后退出
// 搞了一个停止信号，但不知道对不对，因为有个disconnected状态，会不会用指数等待的sleep会更好
func sdpHandler(c *connWithMu, ch chan []byte, p *webrtc.PeerConnection, src *string, dst *string, isClient bool, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case rawMsg, ok := <-ch:
			// 按道理应该sdpHandler先退出再关闭ch
			if !ok {
				return
			}

			var msg common.SignalMsg
			err := json.Unmarshal(rawMsg, &msg)
			if eh.ErrorHandler(err, 1, eh.LogOnly) {
				continue
			}

			switch msg.Type {
			case common.Answer:
				if isClient {
					log.Println("rev a answer msg")
					fromTargetedMsg := common.TargetedMsg{}
					if err := json.Unmarshal([]byte(msg.Data), &fromTargetedMsg); eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}
					answer := webrtc.SessionDescription{}
					if err := json.Unmarshal([]byte(fromTargetedMsg.Data), &answer); eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}

					if err := p.SetRemoteDescription(answer); eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}

					log.Println("set remote desc for answer ok")
				} else {
					log.Println(fmt.Errorf("sdpHandler err: should never receive answer"))
					continue
				}
			case common.Offer:
				if isClient {
					log.Println(fmt.Errorf("sdpHandler err: should never receive offer"))
					continue
				} else {
					log.Println("rev a offer msg")
					fromTargetedMsg := common.TargetedMsg{}
					if err := json.Unmarshal([]byte(msg.Data), &fromTargetedMsg); eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}
					*dst = fromTargetedMsg.Src

					offer := webrtc.SessionDescription{}
					if err := json.Unmarshal([]byte(fromTargetedMsg.Data), &offer); eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}

					if err := p.SetRemoteDescription(offer); eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}

					answer, err := p.CreateAnswer(nil)
					if eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}

					if err := p.SetLocalDescription(answer); eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}

					answerString, err := json.Marshal(answer)
					if eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}

					toTargetedMsg := common.TargetedMsg{
						Src:  *src,
						Dst:  *dst,
						Data: string(answerString),
					}
					targetedString, err := json.Marshal(toTargetedMsg)
					if eh.ErrorHandler(err, 1, eh.LogOnly) {
						continue
					}

					c.writeJSON(&common.SignalMsg{
						Type: common.Answer,
						Data: string(targetedString),
					})
					log.Println("send answer ok")
				}
			case common.Candidate:
				fromTargetedMsg := common.TargetedMsg{}
				if err := json.Unmarshal([]byte(msg.Data), &fromTargetedMsg); eh.ErrorHandler(err, 1, eh.LogOnly) {
					continue
				}

				candidate := webrtc.ICECandidateInit{}
				if err := json.Unmarshal([]byte(fromTargetedMsg.Data), &candidate); eh.ErrorHandler(err, 1, eh.LogOnly) {
					continue
				}

				if err := p.AddICECandidate(candidate); eh.ErrorHandler(err, 1, eh.LogOnly) {
					continue
				}

				log.Println("add ice candidate:", candidate)
			default:
				panic("unhandled default case")
			}
		}
	}
}

func sendRegister(c *websocket.Conn, id string) {
	registerString, err := json.Marshal(common.RegisterMsg{Id: id})
	eh.ErrorHandler(err, 1, eh.LogAndFatal)

	err = c.WriteJSON(&common.SignalMsg{
		Type: common.Register,
		Data: string(registerString),
	})
	eh.ErrorHandler(err, 1, eh.LogAndFatal)
}

func newAndSetOffer(peerConnection *webrtc.PeerConnection) webrtc.SessionDescription {
	offer, err := peerConnection.CreateOffer(nil)
	eh.ErrorHandler(err, 1, eh.LogAndFatal)

	err = peerConnection.SetLocalDescription(offer)
	eh.ErrorHandler(err, 1, eh.LogAndFatal)
	return offer
}

func sendOffer(o webrtc.SessionDescription, c *connWithMu, src string, dst string) {
	offerString, err := json.Marshal(o)
	eh.ErrorHandler(err, 1, eh.LogAndFatal)

	targetedMsg := common.TargetedMsg{
		Src:  src,
		Dst:  dst,
		Data: string(offerString),
	}
	targetedString, err := json.Marshal(targetedMsg)
	eh.ErrorHandler(err, 1, eh.LogAndFatal)

	c.writeJSON(&common.SignalMsg{
		Type: common.Offer,
		Data: string(targetedString),
	})

	log.Println("send offer to signaling server ok")
}

func setPeerConnection(p *webrtc.PeerConnection, natSender *NatSender, src *string, dst *string, f func(msg []byte)) {
	p.OnICEConnectionStateChange(newPeerConnectionOnICEConnectionStateChange())
	p.OnConnectionStateChange(newPeerConnectionOnConnectionStateChange(p, natSender, dst))
	p.OnSignalingStateChange(newPeerConnectionOnSignalingStateChange())
	p.OnDataChannel(newPeerConnectionOnDataChannel(natSender, dst, f))
	p.OnICECandidate(newPeerConnectionOnICECandidate(natSender.signalingServerConn, src, dst))
}

func (natSender *NatSender) dial(dst string) *webrtc.DataChannel {
	dc, ok := natSender.Dst2dc.Get(dst)
	succeeded := make(chan bool)
	defer close(succeeded)
	if ok && dc != nil {
		return dc
	} else {
		source := &(natSender.source)
		destination := &dst

		peerConnection := newPeerConnection(natSender.settingEngine, natSender.webrtcConfig)
		setPeerConnection(peerConnection, natSender, source, destination, natSender.onMessage)
		var err error
		dc, err = peerConnection.CreateDataChannel(*source, nil)
		eh.ErrorHandler(err, 1, eh.LogAndFatal)

		dc.OnOpen(newDataChannelOnOpen(dc, peerConnection, natSender, destination, succeeded))
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("OnMessage '%s':", dst)
			natSender.onMessage(msg.Data)
		})

		dc.OnClose(func() {
			log.Println("sender dc closed with state " + dc.ReadyState().String())
		})

		ch := make(chan []byte)
		natSender.Dst2SdpCh.Set(*destination, ch)
		ctx, cancel := context.WithCancel(natSender.Ctx)
		natSender.dst2cancel.Set(*destination, cancel)
		go sdpHandler(natSender.signalingServerConn, ch, peerConnection, source, destination, true, ctx)
		offer := newAndSetOffer(peerConnection)
		sendOffer(offer, natSender.signalingServerConn, *source, *destination)
		select {
		case <-time.After(time.Second * 5):
			log.Println("dial time out")
			eh.ErrorHandler(dc.Close(), 1, eh.LogAndPanic)
			eh.ErrorHandler(peerConnection.Close(), 1, eh.LogAndPanic)
			close(ch)
			natSender.Dst2SdpCh.Del(*destination)
			cnl, _ := natSender.dst2cancel.Get(*destination)
			cnl()
			natSender.dst2cancel.Del(*destination)
			return nil
		case <-succeeded:
			return dc
		}
	}
}

func (natSender *NatSender) Send(dst string, data []byte) {
	dc := natSender.dial(dst)
	if dc == nil || dc.ReadyState() != webrtc.DataChannelStateOpen {
		log.Println("destination is not online")
		return
	}
	eh.ErrorHandler(dc.Send(data), 1, eh.LogAndPanic)
}

func (natSender *NatSender) Listen(onMessage func(msg []byte)) {
	// 这个可以先尝试直连，不行再用server转发，反正打洞的方式不行也是要转发的，但是打洞方式有一个缺点，就是就算要turn转发，它还是p2p的
	if onMessage != nil {
		natSender.onMessage = onMessage
	}
	if natSender.listeningPc != nil {
		panic("Listen is called more than once")
	}
	source := &(natSender.source)
	destination := new(string)

	natSender.listeningPc = newPeerConnection(natSender.settingEngine, natSender.webrtcConfig)
	setPeerConnection(natSender.listeningPc, natSender, source, destination, natSender.onMessage)
	ctx, cancel := context.WithCancel(natSender.Ctx)
	natSender.listeningCancel = cancel
	go sdpHandler(natSender.signalingServerConn, natSender.listeningSdpCh, natSender.listeningPc, source, destination, false, ctx)
}

func (natSender *NatSender) Accept() {
	select {
	case <-natSender.Ctx.Done():
		return
	case <-natSender.canAccept:
	}
	natSender.listeningPc = nil
	// set to update listener
	destination := new(string)
	source := &natSender.source
	natSender.listeningPc = newPeerConnection(natSender.settingEngine, natSender.webrtcConfig)
	setPeerConnection(natSender.listeningPc, natSender, source, destination, natSender.onMessage)
	natSender.listeningSdpCh = make(chan []byte)
	ctx, cancel := context.WithCancel(natSender.Ctx)
	natSender.listeningCancel = cancel
	go sdpHandler(natSender.signalingServerConn, natSender.listeningSdpCh, natSender.listeningPc, source, destination, false, ctx)
}

func (natSender *NatSender) Close() {
	natSender.signalingServerConn.close()
	natSender.cancel()
	// 等待协程退出
	// 或者用waitGroup？
	time.Sleep(100 * time.Millisecond)
	for _, dc := range natSender.Dst2dc.M {
		eh.ErrorHandler(dc.Close(), 1, eh.LogOnly)
	}
	for _, pc := range natSender.Dst2pc.M {
		eh.ErrorHandler(pc.Close(), 1, eh.LogOnly)
	}
	if natSender.listeningPc != nil {
		eh.ErrorHandler(natSender.listeningPc.Close(), 1, eh.LogOnly)
	}
	close(natSender.listeningSdpCh)
	close(natSender.canAccept)
	for _, ch := range natSender.Dst2SdpCh.M {
		close(ch)
	}
	// cancel自动回收
}

func (natSender *NatSender) GetConnectionType(dst string) (typeString string, ok bool) {
	natSender.dial(dst)
	typeString, ok = natSender.Dst2ConnectionType.Get(dst)
	return
}

// todo:
// dc.Close会关闭本端和对端的dc,触发onclose
// pc.Close会先关闭dc和pc,在把connectionState设置为closed
