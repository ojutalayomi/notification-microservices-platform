import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from 'src/prisma/prisma.service';
import { CreateTemplateDto } from './dto/create-template.dto';
import { Template } from 'generated/prisma';
import { RenderTemplateDto } from './dto/render-template.dto';

export interface PaginationMeta {
  total: number;
  limit: number;
  page: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
}

@Injectable()
export class TemplatesService {
  constructor(private prisma: PrismaService) {}

  async findOne(id: string): Promise<Template> {
    const template = await this.prisma.template.findUnique({
      where: { id },
    });

    if (!template) {
      throw new NotFoundException(`Template with ID ${id} not found`);
    }
    return template;
  }

  async findByName(name: string, language: string = 'en'): Promise<Template> {
    const template = await this.prisma.template.findFirst({
      where: { name, language, is_active: true },
      orderBy: { version: 'desc' },
    });

    if (!template) {
      throw new NotFoundException(
        `Active template with name "${name}" and language "${language}" not found`,
      );
    }
    return template;
  }

  async create(dto: CreateTemplateDto): Promise<Template> {
    const latest: Template | null = await this.prisma.template.findFirst({
      where: { name: dto.name, language: dto.language || 'en' },
      orderBy: { version: 'desc' },
    });

    const version: number = (latest?.version ?? 0) + 1;

    return this.prisma.template.create({
      data: {
        ...dto,
        language: dto.language || 'en',
        version,
        is_active: true,
      },
    });
  }

  async findAll(
    page = 1,
    limit = 10,
  ): Promise<{ data: Template[]; meta: PaginationMeta }> {
    const skip = (page - 1) * limit;
    const [data, total] = await Promise.all([
      this.prisma.template.findMany({
        skip,
        take: limit,
        orderBy: { created_at: 'desc' },
      }),
      this.prisma.template.count(),
    ]);

    const total_pages = Math.ceil(total / limit);

    return {
      data,
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

  async render(dto: RenderTemplateDto): Promise<{
    subject: string;
    html_body: string;
    text_body?: string;
  }> {
    const template: Template | null = await this.prisma.template.findUnique({
      where: { id: dto.template_id, is_active: true },
    });

    if (!template) {
      throw new NotFoundException('Active template not found');
    }

    const missing = template.variables.filter((v) => !(v in dto.variables));
    if (missing.length > 0) {
      throw new BadRequestException(`Missing variables: ${missing.join(', ')}`);
    }

    let subject = template.subject;
    let html_body = template.html_body;
    let text_body = template.text_body;

    for (const [key, value] of Object.entries(dto.variables)) {
      const regex = new RegExp(`{{${key}}}`, 'g');
      subject = subject.replace(regex, value);
      html_body = html_body.replace(regex, value);
      if (text_body) text_body = text_body.replace(regex, value);
    }

    return {
      subject,
      html_body,
      text_body: text_body || undefined,
    };
  }
}
