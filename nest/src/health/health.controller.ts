import { Controller, Get } from "@nestjs/common";
import { PrismaService } from "../prisma/prisma.service";

@Controller("health")
export class HealthController {
  constructor(private readonly prisma: PrismaService) {}

  @Get()
  async check() {
    let isDbConnected = false;

    try {
      await this.prisma.$queryRaw`SELECT 1`;
      isDbConnected = true;
    } catch (error) {
      console.log(error);
      isDbConnected = false;
    }

    return {
      success: true,
      message: "Service is healthy",
      data: {
        status: isDbConnected ? "healthy" : "unhealthy",
        service: "user-service",
        database: isDbConnected ? "connected" : "disconnected",
        timestamp: new Date().toISOString(),
      },
    };
  }
}
