import { Injectable, Logger } from '@nestjs/common';
import { v4 as uuidv4 } from 'uuid';

interface Stream {
  id: string;
  ingestId: string;
  name: string;
  description?: string;
  createdAt: Date;
  subscribers: Set<string>;
  active: boolean;
  metadata?: Record<string, any>;
}

@Injectable()
export class StreamManager {
  private readonly logger = new Logger(StreamManager.name);
  private streams: Map<string, Stream> = new Map();

  constructor() {
    this.logger.log('Stream Manager initialized');
  }

  createStream(name: string, ingestId: string, description?: string, metadata?: Record<string, any>): Stream {
    const id = uuidv4();
    const stream: Stream = {
      id,
      ingestId,
      name,
      description,
      createdAt: new Date(),
      subscribers: new Set<string>(),
      active: false,
      metadata,
    };

    this.streams.set(id, stream);
    this.logger.log(`Created new stream: ${id} - ${name}`);
    return stream;
  }

  getStream(streamId: string): Stream | undefined {
    return this.streams.get(streamId);
  }

  getStreamByIngestId(ingestId: string): Stream | undefined {
    for (const stream of this.streams.values()) {
      if (stream.ingestId === ingestId) {
        return stream;
      }
    }
    return undefined;
  }

  getAllStreams(): Stream[] {
    return Array.from(this.streams.values());
  }

  getActiveStreams(): Stream[] {
    return Array.from(this.streams.values()).filter(stream => stream.active);
  }

  activateStream(streamId: string): boolean {
    const stream = this.streams.get(streamId);
    if (stream) {
      stream.active = true;
      this.logger.log(`Stream activated: ${streamId}`);
      return true;
    }
    return false;
  }

  deactivateStream(streamId: string): boolean {
    const stream = this.streams.get(streamId);
    if (stream) {
      stream.active = false;
      this.logger.log(`Stream deactivated: ${streamId}`);
      return true;
    }
    return false;
  }

  addSubscriber(streamId: string, subscriberId: string): boolean {
    const stream = this.streams.get(streamId);
    if (stream) {
      stream.subscribers.add(subscriberId);
      this.logger.log(`Added subscriber ${subscriberId} to stream ${streamId}`);
      return true;
    }
    return false;
  }

  removeSubscriber(streamId: string, subscriberId: string): boolean {
    const stream = this.streams.get(streamId);
    if (stream) {
      const result = stream.subscribers.delete(subscriberId);
      if (result) {
        this.logger.log(`Removed subscriber ${subscriberId} from stream ${streamId}`);
      }
      return result;
    }
    return false;
  }

  getSubscribers(streamId: string): string[] {
    const stream = this.streams.get(streamId);
    if (stream) {
      return Array.from(stream.subscribers);
    }
    return [];
  }

  deleteStream(streamId: string): boolean {
    const result = this.streams.delete(streamId);
    if (result) {
      this.logger.log(`Deleted stream: ${streamId}`);
    }
    return result;
  }

  updateStreamMetadata(streamId: string, metadata: Record<string, any>): boolean {
    const stream = this.streams.get(streamId);
    if (stream) {
      stream.metadata = { ...stream.metadata, ...metadata };
      this.logger.log(`Updated metadata for stream: ${streamId}`);
      return true;
    }
    return false;
  }
}