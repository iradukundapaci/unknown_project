import {
  WebSocketGateway,
  SubscribeMessage,
  MessageBody,
  ConnectedSocket,
  WebSocketServer,
  OnGatewayInit,
  OnGatewayConnection,
  OnGatewayDisconnect,
} from '@nestjs/websockets';
import { Server, Socket } from 'socket.io';
import { Logger, Injectable } from '@nestjs/common';
import { WebrtcService } from 'src/webrtc/webrtc.service';
import * as wrtc from 'wrtc';

interface JoinRoomDto {
  room: string;
}

interface LeaveRoomDto {
  room: string;
}

interface OfferDto {
  target: string;
  offer: RTCSessionDescriptionInit;
  streamId?: string;
}

interface AnswerDto {
  target: string;
  answer: RTCSessionDescriptionInit;
  streamId?: string;
}

interface IceCandidateDto {
  target: string;
  candidate: RTCIceCandidateInit;
  streamId?: string;
  isProducer?: boolean;
}

@Injectable()
@WebSocketGateway(8000, {
  cors: {
    origin: '*',
    methods: ['GET', 'POST'],
  },
})
export class SignalingGateway
  implements OnGatewayInit, OnGatewayConnection, OnGatewayDisconnect
{
  private readonly logger = new Logger(SignalingGateway.name);
  private rooms = new Map<string, Set<string>>();

  constructor(private readonly webrtcService: WebrtcService) {}

  @WebSocketServer()
  server: Server;

  afterInit(): void {
    this.logger.log('WebRTC Signaling Server Initialized');
  }

  handleConnection(client: Socket): void {
    this.logger.log(`Client connected: ${client.id}`);
  }

  handleDisconnect(client: Socket): void {
    this.logger.log(`Client disconnected: ${client.id}`);
    
    // Clean up WebRTC connections
    this.webrtcService.disconnectPeer(client.id);
    
    // Remove client from all rooms they were in
    this.rooms.forEach((clients, room) => {
      if (clients.has(client.id)) {
        clients.delete(client.id);
        // Notify others in the room that this peer has left
        client.to(room).emit('peerDisconnected', { peerId: client.id });
      }
    });
  }

  @SubscribeMessage('joinRoom')
  handleJoinRoom(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: JoinRoomDto,
  ): void {
    const { room } = data;
    client.join(room);
    
    // Track room membership
    if (!this.rooms.has(room)) {
      this.rooms.set(room, new Set<string>());
    }
    
    // Fixed TS2532: Object is possibly 'undefined'
    const roomClients = this.rooms.get(room);
    if (roomClients) {
      roomClients.add(client.id);
      
      // Fixed TS2769: Array.from with potentially undefined Set
      const otherClients = Array.from(roomClients)
        .filter(id => id !== client.id);
      
      this.logger.log(`Client ${client.id} joined room ${room}`);
      
      // Tell the client about existing peers and available streams
      client.emit('roomJoined', {
        room,
        peers: otherClients,
        availableStreams: this.webrtcService.getStreamIds()
      });
      
      // Notify others that a new peer has joined
      client.to(room).emit('peerJoined', { peerId: client.id });
    }
  }

  @SubscribeMessage('leaveRoom')
  handleLeaveRoom(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: LeaveRoomDto,
  ): void {
    const { room } = data;
    client.leave(room);
    
    // Update room tracking
    const roomClients = this.rooms.get(room);
    if (roomClients) {
      roomClients.delete(client.id);
      // Notify others in the room
      client.to(room).emit('peerDisconnected', { peerId: client.id });
    }
    
    this.logger.log(`Client ${client.id} left room ${room}`);
  }

  @SubscribeMessage('createProducer')
  async handleCreateProducer(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { streamId: string },
  ): Promise<{ offer: RTCSessionDescriptionInit }> {
    try {
      const offer = await this.webrtcService.createProducerPeerConnection(
        data.streamId,
        client.id
      );
      return { offer: offer };
    } catch (error) {
      this.logger.error(`Error creating producer: ${error.message}`);
      throw error;
    }
  }

  @SubscribeMessage('producerAnswer')
  async handleProducerAnswer(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { streamId: string, answer: RTCSessionDescriptionInit }
  ): Promise<void> {
      try {
          const rtcSessionDescription = new wrtc.RTCSessionDescription(data.answer);
          await this.webrtcService.handleProducerAnswer(data.streamId, rtcSessionDescription);
  
          // Notify all clients that the stream is available
          this.server.emit('newStreamAvailable', { streamId: data.streamId });
      } catch (error) {
          this.logger.error(`Error handling producer answer: ${error.message}`);
          throw error;
      }
  }

  @SubscribeMessage('createConsumer')
  async handleCreateConsumer(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { streamId: string },
  ): Promise<{ offer: RTCSessionDescriptionInit | null } | { error: string }> {
    try {
      const offer = await this.webrtcService.createConsumerPeerConnection(
        data.streamId,
        client.id
      );
      
      if (!offer) {
        return { error: 'Stream not available' };
      }
      
      return { offer: offer };
    } catch (error) {
      this.logger.error(`Error creating consumer: ${error.message}`);
      throw error;
    }
  }

  @SubscribeMessage('consumerAnswer')
  async handleConsumerAnswer(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { streamId: string, answer: RTCSessionDescriptionInit },
  ): Promise<void> {
    try {
      // Create RTCSessionDescription object to match the expected type
      const rtcSessionDescription = new wrtc.RTCSessionDescription(data.answer);
      await this.webrtcService.handleConsumerAnswer(
        data.streamId,
        client.id,
        rtcSessionDescription
      );
    } catch (error) {
      this.logger.error(`Error handling consumer answer: ${error.message}`);
      throw error;
    }
  }

  @SubscribeMessage('iceCandidate')
  async handleIceCandidate(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: IceCandidateDto,
  ): Promise<void> {
    try {
      if (!data.streamId) {
        // Traditional direct peer-to-peer forwarding for signaling
        this.server.to(data.target).emit('iceCandidate', {
          candidate: data.candidate,
          from: client.id
        });
      } else {
        // Create RTCIceCandidate object to match the expected type
        const rtcIceCandidate = new wrtc.RTCIceCandidate(data.candidate);
        await this.webrtcService.handleIceCandidate(
          data.streamId,
          client.id,
          rtcIceCandidate,
          data.isProducer
        );
      }
    } catch (error) {
      this.logger.error(`Error handling ICE candidate: ${error.message}`);
      throw error;
    }
  }
  @SubscribeMessage('offer')
  handleOffer(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: OfferDto,
  ): void {
    this.logger.log(`Sending offer from ${client.id} to ${data.target}`);
    // Forward the offer to the intended recipient
    this.server.to(data.target).emit('offer', {
      offer: data.offer,
      from: client.id,
      streamId: data.streamId
    });
  }

  @SubscribeMessage('answer')
  handleAnswer(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: AnswerDto,
  ): void {
    this.logger.log(`Sending answer from ${client.id} to ${data.target}`);
    // Forward the answer to the intended recipient
    this.server.to(data.target).emit('answer', {
      answer: data.answer,
      from: client.id,
      streamId: data.streamId
    });
  }
}