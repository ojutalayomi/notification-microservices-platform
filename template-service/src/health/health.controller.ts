import { Controller, Get } from '@nestjs/common';

@Controller('health')
export class HealthController {
  @Get()
  check() {
    return {
      status: 'ok',
      service: 'template-service',
      timeStamp: new Date().toISOString(),
    };
  }
}
