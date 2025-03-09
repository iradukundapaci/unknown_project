import { Module } from '@nestjs/common';

import { SignalingGateway } from './signaling/signaling.gateway';
import { WebrtcService } from './webrtc/webrtc.service';

@Module({
  providers: [SignalingGateway, WebrtcService],
})
export class AppModule {}
