import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import * as mediasoup from 'mediasoup';
import { Worker, Router, WebRtcTransport, Producer, Consumer, TransportOptions } from 'mediasoup/node/lib/types';
import { ConfigService } from '@nestjs/config';

@Injectable()
export class WebrtcService implements OnModuleInit {
  private readonly logger = new Logger(WebrtcService.name);
  private workers: Worker[] = [];
  private routers: Map<string, Router> = new Map();
  private transports: Map<string, WebRtcTransport> = new Map();
  private producers: Map<string, Producer> = new Map();
  private consumers: Map<string, Consumer> = new Map();
  
  // Map stream IDs to router IDs
  private streamRouters: Map<string, string> = new Map();
  
  constructor(private configService: ConfigService) {}

  async onModuleInit() {
    await this.createWorkers();
  }

  private async createWorkers() {
    const numWorkers = Number(this.configService.get('MEDIASOUP_WORKERS', 1));
    
    for (let i = 0; i < numWorkers; i++) {
      const worker = await mediasoup.createWorker({
        logLevel: 'warn',
        rtcMinPort: Number(this.configService.get('MEDIASOUP_MIN_PORT', 10000)),
        rtcMaxPort: Number(this.configService.get('MEDIASOUP_MAX_PORT', 10100)),
      });

      worker.on('died', () => {
        this.logger.error(`Worker died, exiting: ${worker.pid}`);
        process.exit(1);
      });

      this.workers.push(worker);
      this.logger.log(`Created mediasoup Worker ${i + 1}/${numWorkers}`);
    }
  }

  getWorker(): Worker {
    // Simple round-robin
    const worker = this.workers[0];
    this.workers.push(this.workers.shift());
    return worker;
  }

  async createRouter(streamId: string): Promise<Router> {
    const worker = this.getWorker();
    
    const mediaCodecs = [
      {
        kind: 'audio',
        mimeType: 'audio/opus',
        clockRate: 48000,
        channels: 2,
      },
      {
        kind: 'video',
        mimeType: 'video/VP8',
        clockRate: 90000,
        parameters: {
          'x-google-start-bitrate': 1000,
        },
      },
      {
        kind: 'video',
        mimeType: 'video/H264',
        clockRate: 90000,
        parameters: {
          'packetization-mode': 1,
          'profile-level-id': '42e01f',
          'level-asymmetry-allowed': 1,
        },
      },
    ];

    const router = await worker.createRouter({ mediaCodecs });
    const routerId = router.id;
    this.routers.set(routerId, router);
    this.streamRouters.set(streamId, routerId);
    
    this.logger.log(`Created router ${routerId} for stream ${streamId}`);
    
    return router;
  }

  getRouterForStream(streamId: string): Router | undefined {
    const routerId = this.streamRouters.get(streamId);
    if (routerId) {
      return this.routers.get(routerId);
    }
    return undefined;
  }

  async createWebRtcTransport(routerId: string, isProducer: boolean = false): Promise<WebRtcTransport> {
    const router = this.routers.get(routerId);
    
    if (!router) {
      throw new Error(`Router with ID ${routerId} not found`);
    }

    const transportOptions: TransportOptions = {
      listenIps: [
        {
          ip: this.configService.get('MEDIASOUP_LISTEN_IP', '0.0.0.0'),
          announcedIp: this.configService.get('MEDIASOUP_ANNOUNCED_IP', '127.0.0.1'),
        },
      ],
      initialAvailableOutgoingBitrate: 1000000,
      minimumAvailableOutgoingBitrate: 600000,
      maxSctpMessageSize: 262144,
      enableSctp: true,
      enableTcp: true,
      preferUdp: true,
      appData: { isProducer },
    };

    const transport = await router.createWebRtcTransport(transportOptions);
    this.transports.set(transport.id, transport);
    
    // Handle transport close event
    transport.on('close', () => {
      this.logger.log(`Transport ${transport.id} closed`);
      this.transports.delete(transport.id);
    });
    
    return transport;
  }

  async connectTransport(transportId: string, dtlsParameters: any): Promise<void> {
    const transport = this.transports.get(transportId);
    
    if (!transport) {
      throw new Error(`Transport with ID ${transportId} not found`);
    }
    
    await transport.connect({ dtlsParameters });
    this.logger.log(`Transport ${transportId} connected`);
  }

  async createProducer(transportId: string, rtpParameters: any, kind: 'audio' | 'video'): Promise<Producer> {
    const transport = this.transports.get(transportId);
    
    if (!transport) {
      throw new Error(`Transport with ID ${transportId} not found`);
    }
    
    const producer = await transport.produce({
      kind,
      rtpParameters,
    });
    
    this.producers.set(producer.id, producer);
    
    producer.on('close', () => {
      this.logger.log(`Producer ${producer.id} closed`);
      this.producers.delete(producer.id);
    });
    
    this.logger.log(`Created ${kind} producer ${producer.id} on transport ${transportId}`);
    
    return producer;
  }

  async createConsumer(
    transportId: string, 
    producerId: string, 
    rtpCapabilities: any,
  ): Promise<Consumer | null> {
    const transport = this.transports.get(transportId);
    
    if (!transport) {
      throw new Error(`Transport with ID ${transportId} not found`);
    }
    
    const producer = this.producers.get(producerId);
    
    if (!producer) {
      throw new Error(`Producer with ID ${producerId} not found`);
    }
    
    const router = Array.from(this.routers.values()).find(router => 
      router.canConsume({ producerId, rtpCapabilities })
    );
    
    if (!router) {
      this.logger.warn(`Router cannot consume producer ${producerId}`);
      return null;
    }
    
    const consumer = await transport.consume({
      producerId,
      rtpCapabilities,
      paused: true, // Start in paused mode
    });
    
    this.consumers.set(consumer.id, consumer);
    
    consumer.on('close', () => {
      this.logger.log(`Consumer ${consumer.id} closed`);
      this.consumers.delete(consumer.id);
    });
    
    this.logger.log(`Created consumer ${consumer.id} for producer ${producerId} on transport ${transportId}`);
    
    return consumer;
  }

  async resumeConsumer(consumerId: string): Promise<void> {
    const consumer = this.consumers.get(consumerId);
    
    if (!consumer) {
      throw new Error(`Consumer with ID ${consumerId} not found`);
    }
    
    await consumer.resume();
    this.logger.log(`Resumed consumer ${consumerId}`);
  }

  async closeProducer(producerId: string): Promise<void> {
    const producer = this.producers.get(producerId);
    
    if (producer) {
      producer.close();
      this.logger.log(`Closed producer ${producerId}`);
    }
  }

  async closeConsumer(consumerId: string): Promise<void> {
    const consumer = this.consumers.get(consumerId);
    
    if (consumer) {
      consumer.close();
      this.logger.log(`Closed consumer ${consumerId}`);
    }
  }

  async closeTransport(transportId: string): Promise<void> {
    const transport = this.transports.get(transportId);
    
    if (transport) {
      transport.close();
      this.logger.log(`Closed transport ${transportId}`);
    }
  }

  async closeRouter(routerId: string): Promise<void> {
    const router = this.routers.get(routerId);
    
    if (router) {
      router.close();
      this.routers.delete(routerId);
      
      // Remove stream to router mapping
      for (const [streamId, id] of this.streamRouters.entries()) {
        if (id === routerId) {
          this.streamRouters.delete(streamId);
        }
      }
      
      this.logger.log(`Closed router ${routerId}`);
    }
  }

  getRouterRtpCapabilities(routerId: string): any {
    const router = this.routers.get(routerId);
    
    if (!router) {
      throw new Error(`Router with ID ${routerId} not found`);
    }
    
    return router.rtpCapabilities;
  }

  getTransportStats(transportId: string): Promise<any> {
    const transport = this.transports.get(transportId);
    
    if (!transport) {
      throw new Error(`Transport with ID ${transportId} not found`);
    }
    
    return transport.getStats();
  }

  getProducerStats(producerId: string): Promise<any> {
    const producer = this.producers.get(producerId);
    
    if (!producer) {
      throw new Error(`Producer with ID ${producerId} not found`);
    }
    
    return producer.getStats();
  }

  getConsumerStats(consumerId: string): Promise<any> {
    const consumer = this.consumers.get(consumerId);
    
    if (!consumer) {
      throw new Error(`Consumer with ID ${consumerId} not found`);
    }
    
    return consumer.getStats();
  }
}