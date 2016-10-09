/*
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * Author: FTwOoO <booobooob@gmail.com>
 */

package vpn

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
	"github.com/FTwOoO/vpncore/conn"
	"github.com/FTwOoO/vpncore/tcpip"
	"github.com/FTwOoO/vpncore/tuntap"
	"github.com/FTwOoO/go-logger"
	"github.com/FTwOoO/vpncore/enc"
	"github.com/FTwOoO/link/codec"
	"github.com/FTwOoO/link"
	"github.com/FTwOoO/gohop2/protodef"
	"reflect"
	"github.com/golang/protobuf/proto"
)

const (
	IFACE_BUFSIZE = 2000
	BUF_SIZE = 2048
)

type CandyVPNServer struct {
	cfg       *VPNConfig
	peers     *VPNPeersManager
	iface     *tuntap.Interface
	server    *link.Server

	fromIface chan []byte
	toIface   chan []byte
}

func NewServer(cfg *VPNConfig) (err error) {

	log, err := logger.NewLogger(cfg.LogFile, cfg.LogLevel)
	if err != nil {
		return
	}

	hopServer := new(CandyVPNServer)

	hopServer.fromIface = make(chan []byte, BUF_SIZE)
	hopServer.toIface = make(chan []byte, BUF_SIZE * 4)
	hopServer.peers = new(VPNPeersManager)
	hopServer.cfg = cfg

	iface, err := tuntap.NewTUN("tun1")
	if err != nil {
		return err
	}

	hopServer.iface = iface
	ip, subnet, err := net.ParseCIDR(cfg.Subnet)
	err = iface.SetupNetwork(ip, *subnet, cfg.MTU)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	err = iface.ServerSetupNatRules()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	hopServer.peers = NewVPNPeers(subnet, time.Duration(hopServer.cfg.PeerTimeout) * time.Second)

	go hopServer.listen(cfg.Protocol, enc.Cipher(cfg.Cipher), cfg.Password, fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ServerPort))
	go hopServer.handleInterface()
	go hopServer.forwardFrames()
	go hopServer.peerTimeoutWatcher()

	hopServer.cleanUp()
	return
}

func (srv *CandyVPNServer) handleInterface() {
	go func() {
		for {
			pbytes := <-srv.toIface
			log.Debug("New Net packet to device")
			_, err := srv.iface.Write(pbytes)
			if err != nil {
				return
			}
		}
	}()

	buf := make([]byte, IFACE_BUFSIZE)
	for {
		n, err := srv.iface.Read(buf)
		if err != nil {
			log.Error(err.Error())
			return
		}
		hpbuf := make([]byte, n)
		copy(hpbuf, buf)
		log.Debug("New Net packet from device")
		srv.fromIface <- hpbuf
	}
}

func (srv *CandyVPNServer) listen(protocol conn.TransProtocol, cipher enc.Cipher, pass string, addr string) {
	var err error
	srv.server, err = CreateServer(protocol, addr, cipher, pass, codec.NewProtobufProtocol(srv, []string{}), 0x1000)
	if err != nil {
		log.Errorf("Failed to listen on %s: %s", addr, err.Error())
		os.Exit(0)
	}

	go srv.server.Serve(link.HandlerFunc(sessionLoop))
}

func (srv *CandyVPNServer) forwardFrames() {
	TOP:
	for {
		select {
		case pack := <-srv.fromIface:
			if tcpip.IsIPv4(pack) {
				dest := tcpip.IPv4Packet(pack).DestinationIP().To4()
				log.Debugf("ip dest: %v", dest)
				peer := srv.peers.GetPeerByIp(dest)
				msg := &protodef.Data{
					Header:&protodef.PacketHeader{Sid:peer.Sid, Seq:peer.NextSeq()},
					Payload:pack,
				}

				srv.SendToClient(peer, nil, msg)
			}

		}
	}
}

func sessionLoop(session *link.Session, ctx link.Context, _ error) {
	srv := ctx.(*CandyVPNServer)

	for {
		req, err := session.Receive()
		if err != nil {
			log.Error(err.Error())
			return
		}

		sid := session.ID()
		peer := srv.peers.GetPeerBySid(sid)

		switch req.(type) {
		case protodef.Handshake:
			if peer == nil {
				peer, err = srv.peers.NewPeer(req.(protodef.Handshake).Header.Sid)
				if err != nil {
					log.Errorf("Cant alloc IP from pool %v", err)
				}
				srv.peers.AddSessionToPeer(peer, sid)
			} else {
				srv.peers.AddSessionToPeer(peer, sid)
			}

			peer.LastSeenTime = time.Now()
			log.Debugf("assign address %s", peer.Ip)
			atomic.StoreInt32(&peer.state, HOP_STAT_HANDSHAKE)

			size, _ := srv.peers.IpPool.subnet.Mask.Size()

			msg := &protodef.HandshakeAck{
				Header:req.(protodef.Handshake).Header,
				Ip:peer.Ip,
				MarkSize:size,
			}
			err = srv.SendToClient(peer, session, msg)

			go func() {
				select {
				case <-peer.hsDone:
					peer.state = HOP_STAT_WORKING
					return
				case <-time.After(8 * time.Second):
					msg := &protodef.Fin{Header:req.(protodef.Handshake).Header}
					err = srv.SendToClient(peer, session, msg)
					srv.peers.DeletePeer(peer)
				}
			}()
		case protodef.Ping:
			if peer.state == HOP_STAT_WORKING {

				msg := &protodef.PingAck{Header:req.(protodef.Ping).Header}
				err = srv.SendToClient(peer, session, msg)
			}
		case protodef.Data:
			if peer.state == HOP_STAT_WORKING {
				srv.toIface <- req.(protodef.Data).Payload
			}
		case protodef.Fin:
			log.Infof("Releasing client ip: %d", peer.Ip)
			msg := &protodef.FinAck{Header:req.(protodef.Fin).Header}
			err = srv.SendToClient(peer, session, msg)
			srv.peers.DeletePeer(peer)

		case protodef.HandshakeAck:
			log.Infof("Client %d Connected", peer.Ip)
			if ok := atomic.CompareAndSwapInt32(&peer.state, HOP_STAT_HANDSHAKE, HOP_STAT_WORKING); ok {
				close(peer.hsDone)
			}
		case protodef.PingAck:
			return
		case protodef.DataAck:
		case protodef.FinAck:
		default:
			log.Errorf("Message type %s that server dont support yet!\n", reflect.TypeOf(req))
		}
	}
}

func (srv *CandyVPNServer) cleanUp() {

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT)
	<-c

	allPeers := srv.peers.GetAllPeers()

	for _, peer := range allPeers {
		msg := &protodef.Fin{
			Header:&protodef.PacketHeader{Sid:peer.Sid, Seq:peer.NextSeq()},
		}
		srv.SendToClient(peer, nil, msg)
	}
	os.Exit(0)
}

func (srv *CandyVPNServer) peerTimeoutWatcher() {
	timeout := time.Second * time.Duration(srv.cfg.PeerTimeout)

	for {
		select {
		case <-time.After(timeout):
			allPeers := srv.peers.GetAllPeers()
			for _, peer := range allPeers {
				log.Debugf("IP: %v", peer.Ip)
				msg := &protodef.Ping{
					Header:&protodef.PacketHeader{Sid:peer.Sid, Seq:peer.NextSeq()},
				}
				srv.SendToClient(peer, nil, msg)
			}
		case peer := <-srv.peers.PeerTimeout:
			msg := &protodef.Fin{
				Header:&protodef.PacketHeader{Sid:peer.Sid, Seq:peer.NextSeq()},
			}
			srv.SendToClient(peer, nil, msg)
		}
	}
}

func (srv *CandyVPNServer) SendToClient(peer *VPNPeer, session *link.Session, msg proto.Message) error {
	if session != nil {
		return session.Send(msg)
	}

	allSessionId := srv.peers.GetPeerSessions(peer)

	for _, sessionId := range allSessionId {
		session := srv.server.GetSession(sessionId)
		if session != nil {
			err := session.Send(msg)
			if err != nil {
				log.Error(err.Error())
				continue
			}
			return nil
		}
	}

	return fmt.Errorf("Send msg fail: %v", msg)
}