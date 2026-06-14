package webrtc

import (
	"fmt"

	pion "github.com/pion/webrtc/v4"
)

type peer struct {
	clientID string
	pc       *pion.PeerConnection
}

func newPeer(clientID string, onICE func(candidate string)) (*peer, error) {
	mediaEngine := &pion.MediaEngine{}
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("register codecs: %w", err)
	}

	api := pion.NewAPI(pion.WithMediaEngine(mediaEngine))

	pc, err := api.NewPeerConnection(pion.Configuration{
		ICEServers: []pion.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("peer connection: %w", err)
	}

	p := &peer{clientID: clientID, pc: pc}

	pc.OnTrack(func(track *pion.TrackRemote, _ *pion.RTPReceiver) {
		go discardTrack(track)
	})

	pc.OnICECandidate(func(c *pion.ICECandidate) {
		if c == nil || onICE == nil {
			return
		}
		init := c.ToJSON()
		onICE(mustMarshalICE(init))
	})

	return p, nil
}

func (p *peer) setRemoteOfferAndCreateAnswer(sdp string) (string, error) {
	offer := pion.SessionDescription{
		Type: pion.SDPTypeOffer,
		SDP:  sdp,
	}
	if err := p.pc.SetRemoteDescription(offer); err != nil {
		return "", fmt.Errorf("set remote offer: %w", err)
	}
	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("create answer: %w", err)
	}
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("set local answer: %w", err)
	}
	return answer.SDP, nil
}

func (p *peer) addICE(candidateJSON string) error {
	init, err := parseICE(candidateJSON)
	if err != nil {
		return err
	}
	return p.pc.AddICECandidate(init)
}

func (p *peer) close() error {
	return p.pc.Close()
}

func discardTrack(track *pion.TrackRemote) {
	buf := make([]byte, 1500)
	for {
		if _, _, err := track.Read(buf); err != nil {
			return
		}
	}
}
