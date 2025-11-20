import {
  Injectable,
  NotFoundException,
  BadRequestException,
  Logger,
  HttpException,
} from "@nestjs/common";
import { HttpService } from "@nestjs/axios";
import { RabbitMQService } from "../rabbitmq/rabbitmq.service";
import {
  SendNotificationDto,
  NotificationType,
} from "./dto/send-notification.dto";
import { v4 as uuidv4 } from "uuid";
import { firstValueFrom } from "rxjs";

@Injectable()
export class NotificationsService {
  private readonly logger = new Logger(NotificationsService.name);
  private notificationStore = new Map(); // In-memory store (use Redis in production)

  constructor(
    private readonly rabbitMQService: RabbitMQService,
    private readonly httpService: HttpService,
  ) {}

  async sendNotification(dto: SendNotificationDto) {
    const notification_id = uuidv4();

    try {
      // 1. Fetch user data from User Service
      this.logger.log(`Fetching user data for user_id: ${dto.user_id}`);
      const user = await this.getUserData(dto.user_id);

      if (!user) {
        throw new NotFoundException(`User with ID ${dto.user_id} not found`);
      }

      // 2. Check user preferences
      if (dto.type === NotificationType.EMAIL && !user.email_enabled) {
        throw new BadRequestException("User has disabled email notifications");
      }

      if (dto.type === NotificationType.PUSH && !user.push_enabled) {
        throw new BadRequestException("User has disabled push notifications");
      }

      // 3. Fetch template (if needed)
      const template = await this.getTemplate(dto.template_id);

      // 4. Prepare message payload
      const basePayload = {
        notification_id,
        user_id: dto.user_id,
        template_id: dto.template_id,
        data: dto.data || {},
        priority: dto.priority || "normal",
        timestamp: new Date().toISOString(),
      };

      // 5. Send to appropriate queue(s)
      if (
        dto.type === NotificationType.EMAIL ||
        dto.type === NotificationType.BOTH
      ) {
        const emailPayload = {
          ...basePayload,
          email: user.email,
          name: user.name,
          template,
        };
        this.rabbitMQService.publishToQueue("email", emailPayload);
      }

      if (
        dto.type === NotificationType.PUSH ||
        dto.type === NotificationType.BOTH
      ) {
        const pushPayload = {
          ...basePayload,
          push_token: user.push_token,
          name: user.name,
          template: renderedTemplate,
        };
        this.rabbitMQService.publishToQueue("push", pushPayload);
      }

      // 6. Store notification status
      const notificationStatus = {
        notification_id,
        user_id: dto.user_id,
        type: dto.type,
        status: "queued",
        created_at: new Date(),
      };

      this.notificationStore.set(notification_id, notificationStatus);

      this.logger.log(`✅ Notification ${notification_id} queued successfully`);

      return {
        message: "Notification queued successfully",
        data: notificationStatus,
      };
    } catch (error) {
      this.logger.error(`❌ Error sending notification:`, error);

      // Store failed notification
      const failedNotification = {
        notification_id,
        user_id: dto.user_id,
        type: dto.type,
        status: "failed",
        error: error.message,
        created_at: new Date(),
      };

      this.notificationStore.set(notification_id, failedNotification);

      // Send to dead letter queue
      this.rabbitMQService.publishToQueue("failed", failedNotification);

      if (error instanceof HttpException) {
        throw error;
      }

      throw new BadRequestException("Failed to send notification");
    }
  }

  getNotificationStatus(notification_id: string) {
    const notification = this.notificationStore.get(notification_id);

    if (!notification) {
      throw new NotFoundException(
        `Notification with ID ${notification_id} not found`,
      );
    }

    return {
      message: "Notification status retrieved",
      data: notification,
    };
  }

  getAllNotifications(page: number = 1, limit: number = 10) {
    const notifications = Array.from(this.notificationStore.values());
    const total = notifications.length;
    const start = (page - 1) * limit;
    const end = start + limit;
    const paginatedData = notifications.slice(start, end);

    const total_pages = Math.ceil(total / limit);

    return {
      message: "Notifications retrieved successfully",
      data: paginatedData,
      meta: {
        total,
        limit,
        page,
        total_pages,
        has_next: page < total_pages,
        has_previous: page > 1,
      },
    };
  }

  // Helper: Fetch user data from User Service
  private async getUserData(user_id: string) {
    try {
      const url = `${process.env.USER_SERVICE_URL}/users/${user_id}`;
      console.log(url);
      this.logger.log(`Calling User Service: ${url}`);

      const response = await firstValueFrom(this.httpService.get(url));
      return response.data.data;
    } catch (error) {
      this.logger.error("Error fetching user data:", error.message);
      throw new NotFoundException(`User with ID ${user_id} not found`);
    }
  }

  // Helper: Fetch template from Template Service
  private async getTemplate(template_id: string) {
    try {
      const url = `${process.env.TEMPLATE_SERVICE_URL}/templates/${template_id}`;
      this.logger.log(`Calling Template Service: ${url}`);

      const response = await firstValueFrom(this.httpService.get(url));
      return response.data.data;
    } catch (error) {
      // Check if it's a 404 error (template not found)
      const isNotFound =
        error?.response?.status === 404 ||
        error?.status === 404 ||
        (error instanceof HttpException && error.getStatus() === 404);

      if (isNotFound) {
        this.logger.warn(
          `Template ${template_id} not found in Template Service, using default template`,
        );
      } else {
        // Log other errors (network issues, service unavailable, etc.)
        this.logger.error(
          `Error fetching template ${template_id} from Template Service: ${error?.message || error}`,
        );
        this.logger.warn(
          `Falling back to default template for ${template_id} due to service error`,
        );
      }

      // Return a simple default template
      return {
        template_id,
        subject: "Notification",
        body: "You have a new notification",
      };
    }
  }
}
