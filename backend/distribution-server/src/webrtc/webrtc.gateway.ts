import { 
  WebSocketGateway, 
  WebSocketServer,
  SubscribeMessage, 
  OnGatewayConnection, 
  OnGatewayDisconnect,
  ConnectedSocket,
  MessageBody
} from '@nestjs/websockets';
import { Logger } from '@nestjs/common';
import { Server, Socket } from 'socket.io';
import { WebrtcService } from './webrtc.service';
import { StreamManager } from './stream-manager.service';

interface ConnectedClient {
  id: string;
  socket: Socket;
  producerTransports: Map<string, string>; // key: streamId, value: transportId
  consumerTransports: Map<string, string>; // key: streamId, value: transportId
  producers: Map<string, string[]>; // key: streamId, value: array of producerIds
  consumers: Map<string, string[]>; // key: streamId, value: array of consumerIds
  role: 'ingest' | 'subscriber';
}

@WebSocketGateway({
  cors: {
    origin: '*',
  },
  namespace: '/webrtc',
})
export class WebrtcGateway implements OnGatewayConnection, OnGatewayDisconnect {
  @WebSocketServer() server: Server;
  
  private readonly logger = new Logger(WebrtcGateway.name);
  private clients: Map<string, ConnectedClient> = new Map();
  
  constructor(
    private readonly webrtcService: WebrtcService,
    private readonly streamManager: StreamManager,
  ) {}

  afterInit() {
    this.logger.log('WebRTC Gateway initialized');
  }

  handleConnection(client: Socket) {
    this.logger.log(`Client connected: ${client.id}`);
    
    const newClient: ConnectedClient = {
      id: client.id,
      socket: client,
      producerTransports: new Map(),
      consumerTransports: new Map(),
      producers: new Map(),
      consumers: new Map(),
      role: 'subscriber', // Default role
    };
    
    this.clients.set(client.id, newClient);
  }

  async handleDisconnect(client: Socket) {
    this.logger.log(`Client disconnected: ${client.id}`);
    
    const connectedClient = this.clients.get(client.id);
    
    if (connectedClient) {
      // Close all producer transports
      for (const [streamId, transportId] of connectedClient.producerTransports.entries()) {
        try {
          await this.webrtcService.closeTransport(transportId);
          
          // If this was an ingest client, deactivate the stream
          if (connectedClient.role === 'ingest') {
            this.streamManager.deactivateStream(streamId);
            
            // Notify all subscribers that the stream is no longer active
            this.server.emit('stream-deactivated', { streamId });
          }
        } catch (error) {
          this.logger.error(`Error closing producer transport: ${error.message}`);
        }
      }
      
      // Close all consumer transports
      for (const [, transportId] of connectedClient.consumerTransports.entries()) {
        try {
          await this.webrtcService.closeTransport(transportId);
        } catch (error) {
          this.logger.error(`Error closing consumer transport: ${error.message}`);
        }
      }
      
      // Remove subscriber from all streams
      for (const stream of this.streamManager.getAllStreams()) {
        this.streamManager.removeSubscriber(stream.id, client.id);
      }
      
      this.clients.delete(client.id);
    }
  }

  @SubscribeMessage('register-as-ingest')
  async handleRegisterAsIngest(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { streamName: string, description?: string, metadata?: Record<string, any> },
  ) {
    const connectedClient = this.clients.get(client.id);
    
    if (!connectedClient) {
      return { error: 'Client not found' };
    }
    
    // Update role
    connectedClient.role = 'ingest';
    
    // Create a new stream
    const stream = this.streamManager.createStream(
      data.streamName,
      client.id,
      data.description,
      data.metadata,
    );
    
    // Create a router for this stream
    const router = await this.webrtcService.createRouter(stream.id);
    
    return {
      streamId: stream.id,
      routerId: router.id,
      rtpCapabilities: router.rtpCapabilities,
    };
  }

  @SubscribeMessage('get-streams')
  handleGetStreams() {
    const activeStreams = this.streamManager.getActiveStreams().map(stream => ({
      id: stream.id,
      name: stream.name,
      description: stream.description,
      active: stream.active,
      createdAt: stream.createdAt,
      subscriberCount: stream.subscribers.size,
    }));
    
    return { streams: activeStreams };
  }

  @SubscribeMessage('get-stream')
  handleGetStream(@MessageBody() data: { streamId: string }) {
    const stream = this.streamManager.getStream(data.streamId);
    
    if (!stream) {
      return { error: 'Stream not found' };
    }
    
    return {
      stream: {
        id: stream.id,
        name: stream.name,
        description: stream.description,
        active: stream.active,
        createdAt: stream.createdAt,
        subscriberCount: stream.subscribers.size,
      },
    };
  }

  @SubscribeMessage('get-router-capabilities')
  handleGetRouterCapabilities(@MessageBody() data: { streamId: string }) {
    const router = this.webrtcService.getRouterForStream(data.streamId);
    
    if (!router) {
      return { error: 'Router not found for stream' };
    }
    
    return { rtpCapabilities: router.rtpCapabilities };
  }

  @SubscribeMessage('create-producer-transport')
  async handleCreateProducerTransport(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { streamId: string },
  ) {
    const connectedClient = this.clients.get(client.id);
    
    if (!connectedClient) {
      return { error: 'Client not found' };
    }
    
    const stream = this.streamManager.getStream(data.streamId);
    
    if (!stream) {
      return { error: 'Stream not found' };
    }
    
    const router = this.webrtcService.getRouterForStream(data.streamId);
    
    if (!router) {
      return { error: 'Router not found for stream' };
    }
    
    try {
      const transport = await this.webrtcService.createWebRtcTransport(router.id, true);
      
      // Store the transport ID for this stream
      connectedClient.producerTransports.set(data.streamId, transport.id);
      
      return {
        id: transport.id,
        iceParameters: transport.iceParameters,
        iceCandidates: transport.iceCandidates,
        dtlsParameters: transport.dtlsParameters,
        sctpParameters: transport.sctpParameters,
      };
    } catch (error) {
      this.logger.error(`Error creating producer transport: ${error.message}`);
      return { error: error.message };
    }
  }

  @SubscribeMessage('connect-producer-transport')
  async handleConnectProducerTransport(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { transportId: string, dtlsParameters: any },
  ) {
    try {
      await this.webrtcService.connectTransport(data.transportId, data.dtlsParameters);
      return { success: true };
    } catch (error) {
      this.logger.error(`Error connecting producer transport: ${error.message}`);
      return { error: error.message };
    }
  }

  @SubscribeMessage('produce')
  async handleProduce(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { 
      transportId: string, 
      kind: 'audio' | 'video', 
      rtpParameters: any,
      streamId: string,
    },
  ) {
    const connectedClient = this.clients.get(client.id);
    
    if (!connectedClient) {
      return { error: 'Client not found' };
    }
    
    try {
      const producer = await this.webrtcService.createProducer(
        data.transportId,
        data.rtpParameters,
        data.kind,
      );
      
      // Store the producer ID for this stream
      if (!connectedClient.producers.has(data.streamId)) {
        connectedClient.producers.set(data.streamId, []);
      }
      connectedClient.producers.get(data.streamId).push(producer.id);
      
      // Activate the stream if it's the first producer
      if (!this.streamManager.getStream(data.streamId).active) {
        this.streamManager.activateStream(data.streamId);
        
        // Notify all clients that a new stream is available
        this.server.emit('stream-activated', { 
          streamId: data.streamId,
          producerId: producer.id,
          kind: data.kind,
        });
      } else {
        // Notify all subscribers of this stream that there's a new producer
        const subscribers = this.streamManager.getSubscribers(data.streamId);
        
        for (const subscriberId of subscribers) {
          const subscriberSocket = this.clients.get(subscriberId)?.socket;
          
          if (subscriberSocket) {
            subscriberSocket.emit('new-producer', {
              streamId: data.streamId,
              producerId: producer.id,
              kind: data.kind,
            });
          }
        }
      }
      
      return { id: producer.id };
    } catch (error) {
      this.logger.error(`Error producing: ${error.message}`);
      return { error: error.message };
    }
  }

  @SubscribeMessage('subscribe-to-stream')
  async handleSubscribeToStream(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { streamId: string, rtpCapabilities: any },
  ) {
    const connectedClient = this.clients.get(client.id);
    
    if (!connectedClient) {
      return { error: 'Client not found' };
    }
    
    const stream = this.streamManager.getStream(data.streamId);
    
    if (!stream) {
      return { error: 'Stream not found' };
    }
    
    if (!stream.active) {
      return { error: 'Stream is not active' };
    }
    
    // Add the client as a subscriber
    this.streamManager.addSubscriber(data.streamId, client.id);
    
    const router = this.webrtcService.getRouterForStream(data.streamId);
    
    if (!router) {
      return { error: 'Router not found for stream' };
    }
    
    // Find the ingest client for this stream
    const ingestClient = Array.from(this.clients.values()).find(
      c => c.role === 'ingest' && c.producerTransports.has(data.streamId)
    );
    
    if (!ingestClient) {
      return { error: 'Ingest client not found for stream' };
    }
    
    // Get all producers for this stream
    const producerIds = ingestClient.producers.get(data.streamId) || [];
    
    if (producerIds.length === 0) {
      return { error: 'No producers found for stream' };
    }
    
    return {
      success: true,
      producerIds,
      routerId: router.id,
      rtpCapabilities: router.rtpCapabilities,
    };
  }

  @SubscribeMessage('create-consumer-transport')
  async handleCreateConsumerTransport(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { streamId: string },
  ) {
    const connectedClient = this.clients.get(client.id);
    
    if (!connectedClient) {
      return { error: 'Client not found' };
    }
    
    const stream = this.streamManager.getStream(data.streamId);
    
    if (!stream) {
      return { error: 'Stream not found' };
    }
    
    const router = this.webrtcService.getRouterForStream(data.streamId);
    
    if (!router) {
      return { error: 'Router not found for stream' };
    }
    
    try {
      const transport = await this.webrtcService.createWebRtcTransport(router.id, false);
      
      // Store the transport ID for this stream
      connectedClient.consumerTransports.set(data.streamId, transport.id);
      
      return {
        id: transport.id,
        iceParameters: transport.iceParameters,
        iceCandidates: transport.iceCandidates,
        dtlsParameters: transport.dtlsParameters,
        sctpParameters: transport.sctpParameters,
      };
    } catch (error) {
      this.logger.error(`Error creating consumer transport: ${error.message}`);
      return { error: error.message };
    }
  }

  @SubscribeMessage('connect-consumer-transport')
  async handleConnectConsumerTransport(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { transportId: string, dtlsParameters: any },
  ) {
    try {
      await this.webrtcService.connectTransport(data.transportId, data.dtlsParameters);
      return { success: true };
    } catch (error) {
      this.logger.error(`Error connecting consumer transport: ${error.message}`);
      return { error: error.message };
    }
  }

  @SubscribeMessage('consume')
  async handleConsume(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { 
      transportId: string, 
      producerId: string, 
      rtpCapabilities: any,
      streamId: string,
    },
  ) {
    const connectedClient = this.clients.get(client.id);
    
    if (!connectedClient) {
      return { error: 'Client not found' };
    }
    
    try {
      const consumer = await this.webrtcService.createConsumer(
        data.transportId,
        data.producerId,
        data.rtpCapabilities,
      );
      
      if (!consumer) {
        return { error: 'Could not create consumer' };
      }
      
      // Store the consumer ID for this stream
      if (!connectedClient.consumers.has(data.streamId)) {
        connectedClient.consumers.set(data.streamId, []);
      }
      connectedClient.consumers.get(data.streamId).push(consumer.id);
      
      return {
        id: consumer.id,
        producerId: data.producerId,
        kind: consumer.kind,
        rtpParameters: consumer.rtpParameters,
        type: consumer.type,
      };
    } catch (error) {
      this.logger.error(`Error consuming: ${error.message}`);
      return { error: error.message };
    }
  }

  @SubscribeMessage('resume-consumer')
  async handleResumeConsumer(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { consumerId: string },
  ) {
    try {
      await this.webrtcService.resumeConsumer(data.consumerId);
      return { success: true };
    } catch (error) {
      this.logger.error(`Error resuming consumer: ${error.message}`);
      return { error: error.message };
    }
  }

  @SubscribeMessage('pause-consumer')
  async handlePauseConsumer(
    @ConnectedSocket() client: Socket,
    @MessageBody() data: { consumerId: string },
  ) {
    const consumer = this.webrtcService.getConsumer(data.consumerId);
    
    if (!consumer) {
      return { error: 'Consumer not found' };
    }
    
    try {
      await consumer.pause();
      return { success: true };
    } catch (error) {
      this.logger.error(`Error pausing consumer: ${error.message}`);
      return { error: error.message };
    }
  }

  @SubscribeMessage('get-producer-stats')
  async handleGetProducerStats(
    @MessageBody() data: { producerId: string },
  ) {
    try {
      const stats = await this.webrtcService.getProducerStats(data.producerId);
      return { stats };
    } catch (error) {
      return { error: error.message };
    }
  }

  @SubscribeMessage('get-consumer-stats')
  async handleGetConsumerStats(
    @MessageBody() data: { consumerId: string },
  ) {
    try {
      const stats = await this.webrtcService.getConsumerStats(data.consumerId);
      return { stats };
    } catch (error) {
      return { error: error.message };
    }
  }

  @SubscribeMessage('get-transport-stats')
  async handleGetTransportStats(
    @MessageBody() data: { transportId: string },
  ) {
    try {
      const stats = await this.webrtcService.getTransportStats(data.transportId);
      return { stats };
    } catch (error) {
      return { error: error.message };
    }
  }
}