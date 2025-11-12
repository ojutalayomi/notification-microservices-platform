import { Body, Post, Query, Controller, Get, Param } from '@nestjs/common';
import { TemplatesService } from './templates.service';
import { CreateTemplateDto } from './dto/create-template.dto';
import { RenderTemplateDto } from './dto/render-template.dto';

@Controller('templates')
export class TemplatesController {
  constructor(private readonly service: TemplatesService) {}

  @Get(':id')
  async findOne(@Param('id') id: string) {
    const data = await this.service.findOne(id);
    return {
      success: true,
      data,
      message: 'Template fetched successfully',
    };
  }

  @Post('render')
  async render(@Body() dto: RenderTemplateDto) {
    const data = await this.service.render(dto);
    return {
      success: true,
      data,
      message: 'Template rendered successfully',
    };
  }

  @Post()
  async create(@Body() dto: CreateTemplateDto) {
    const data = await this.service.create(dto);
    return {
      success: true,
      data,
      message: 'Template created successfully',
    };
  }

  @Get()
  async findAll(@Query('page') page = 1, @Query('limit') limit = 10) {
    const { data, meta } = await this.service.findAll(+page, +limit);
    return {
      success: true,
      data,
      message: 'Templates fetched successfully',
      meta,
    };
  }
}
