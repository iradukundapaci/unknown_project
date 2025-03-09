import { Injectable, Logger } from '@nestjs/common';
import * as wrtc from 'wrtc';

interface PeerConnection {
  id: string;
  connection: RTCPeerConnection;
  stream?: MediaStream;
}

@Injectable()
export class WebrtcService {
  private readonly logger = new Logger(WebrtcService.name);
  private producers = new Map<string, PeerConnection>();
  private consumers = new Map<string, PeerConnection[]>();
  
  constructor() {
    this.logger.log('WebRTC Service Initialized');
  }

  async createProducerPeerConnection(streamId: string, peerId: string): Promise<RTCSessionDescription> {
    this.logger.log(`Creating producer peer connection for stream ${streamId} from peer ${peerId}`);
    
    // Create RTCPeerConnection with STUN servers for NAT traversal
    const peerConnection = new wrtc.RTCPeerConnection({
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' }
      ]
    });
    
    // Store the producer connection
    this.producers.set(streamId, {
      id: peerId,
      connection: peerConnection
    });
    
    // Initialize consumers list for this stream
    this.consumers.set(streamId, []);
    
    // Handle ICE candidates
    peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        this.logger.debug(`New ICE candidate for producer ${peerId}`);
        // In a real implementation, we would send this back to the client
      }
    };
    
    // Handle incoming streams
    peerConnection.ontrack = (event) => {
      this.logger.log(`Received track from producer ${peerId}`);
      const stream = event.streams[0];
      
      // Store the stream to redistribute it
      const producer = this.producers.get(streamId);
      if (producer) {
        producer.stream = stream;
        this.logger.log(`Stream ${streamId} is now available for distribution`);
      }
    };
    
    // Create an offer to receive video
    const offer = await peerConnection.createOffer({
      offerToReceiveVideo: true,
      offerToReceiveAudio: true
    });
    
    await peerConnection.setLocalDescription(offer);
    
    return offer;
  }
  
  async handleProducerAnswer(streamId: string, answer: RTCSessionDescription): Promise<void> {
    const producer = this.producers.get(streamId);
    
    if (!producer) {
      throw new Error(`No producer found for stream ${streamId}`);
    }
    
    await producer.connection.setRemoteDescription(
      new wrtc.RTCSessionDescription(answer)
    );
    
    this.logger.log(`Producer answer processed for stream ${streamId}`);
  }
  
  async createConsumerPeerConnection(streamId: string, consumerId: string): Promise<RTCSessionDescription | null> {
    const producer = this.producers.get(streamId);
    
    if (!producer || !producer.stream) {
      this.logger.warn(`No stream available for ${streamId}`);
      return null;
    }
    
    this.logger.log(`Creating consumer connection for ${consumerId} to stream ${streamId}`);
    
    // Create a new peer connection for the consumer
    const peerConnection = new wrtc.RTCPeerConnection({
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' }
      ]
    });
    
    // Add this consumer to our tracking
    const consumersList = this.consumers.get(streamId) || [];
    consumersList.push({
      id: consumerId,
      connection: peerConnection
    });
    this.consumers.set(streamId, consumersList);
    
    // Add the producer's tracks to this connection
    producer.stream.getTracks().forEach(track => {
      peerConnection.addTrack(track, producer.stream);
    });
    
    // Handle ICE candidates
    peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        this.logger.debug(`New ICE candidate for consumer ${consumerId}`);
        // In a real implementation, we would send this back to the client
      }
    };
    
    // Create an offer to send video
    const offer = await peerConnection.createOffer();
    await peerConnection.setLocalDescription(offer);
    
    return offer;
  }
  
  async handleConsumerAnswer(streamId: string, consumerId: string, answer: RTCSessionDescription): Promise<void> {
    const consumersList = this.consumers.get(streamId);
    
    if (!consumersList) {
      throw new Error(`No consumers found for stream ${streamId}`);
    }
    
    const consumer = consumersList.find(c => c.id === consumerId);
    
    if (!consumer) {
      throw new Error(`Consumer ${consumerId} not found for stream ${streamId}`);
    }
    
    await consumer.connection.setRemoteDescription(
      new wrtc.RTCSessionDescription(answer)
    );
    
    this.logger.log(`Consumer answer processed for ${consumerId} on stream ${streamId}`);
  }
  
  async handleIceCandidate(
    streamId: string, 
    peerId: string, 
    candidate: RTCIceCandidate, 
    isProducer = false
  ): Promise<void> {
    if (isProducer) {
      const producer = this.producers.get(streamId);
      if (producer) {
        await producer.connection.addIceCandidate(new wrtc.RTCIceCandidate(candidate));
      }
    } else {
      const consumersList = this.consumers.get(streamId);
      if (consumersList) {
        const consumer = consumersList.find(c => c.id === peerId);
        if (consumer) {
          await consumer.connection.addIceCandidate(new wrtc.RTCIceCandidate(candidate));
        }
      }
    }
  }
  
  getStreamIds(): string[] {
    return Array.from(this.producers.keys());
  }
  
  disconnectPeer(peerId: string): void {
    // Check if the peer is a producer
    this.producers.forEach((producer, streamId) => {
      if (producer.id === peerId) {
        this.logger.log(`Closing producer connection for ${peerId}`);
        producer.connection.close();
        this.producers.delete(streamId);
        
        // Also close all associated consumer connections
        const consumersList = this.consumers.get(streamId);
        if (consumersList) {
          consumersList.forEach(consumer => consumer.connection.close());
          this.consumers.delete(streamId);
        }
      }
    });
    
    // Check if the peer is a consumer in any stream
    this.consumers.forEach((consumersList, streamId) => {
      const updatedList = consumersList.filter(consumer => {
        if (consumer.id === peerId) {
          this.logger.log(`Closing consumer connection for ${peerId}`);
          consumer.connection.close();
          return false;
        }
        return true;
      });
      
      if (updatedList.length !== consumersList.length) {
        this.consumers.set(streamId, updatedList);
      }
    });
  }
}