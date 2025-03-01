import { Module } from '@nestjs/common';

@Module({
  providers: [WebrtcService, WebrtcGateway, StreamManagerService]
  exports: [WebrtcService, StreamManagerService]
})
export class WebrtcModule {}
