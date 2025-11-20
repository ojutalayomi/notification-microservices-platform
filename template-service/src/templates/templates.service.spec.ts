import { Test, TestingModule } from '@nestjs/testing';
import { TemplatesService } from './templates.service';
import { PrismaService } from '../prisma/prisma.service';
import { NotFoundException, BadRequestException } from '@nestjs/common';
import { CreateTemplateDto } from './dto/create-template.dto';
import { RenderTemplateDto } from './dto/render-template.dto';

describe('TemplatesService', () => {
  let service: TemplatesService;

  const mockPrismaService = {
    template: {
      findUnique: jest.fn(),
      findFirst: jest.fn(),
      create: jest.fn(),
      findMany: jest.fn(),
      count: jest.fn(),
    },
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        TemplatesService,
        {
          provide: PrismaService,
          useValue: mockPrismaService,
        },
      ],
    }).compile();

    service = module.get<TemplatesService>(TemplatesService);

    // Reset all mocks before each test
    jest.clearAllMocks();
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('findOne', () => {
    it('should return a template when found', async () => {
      const templateId = '71a258ea-e6f5-415a-9ca5-b077425f8344';
      const mockTemplate = {
        id: templateId,
        name: 'welcome-email',
        language: 'en',
        subject: 'Welcome {{name}}',
        html_body: '<h1>Welcome {{name}}</h1>',
        text_body: 'Welcome {{name}}',
        variables: ['name'],
        version: 1,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      mockPrismaService.template.findUnique.mockResolvedValue(mockTemplate);

      const result = await service.findOne(templateId);

      expect(result).toEqual(mockTemplate);
      expect(mockPrismaService.template.findUnique).toHaveBeenCalledWith({
        where: { id: templateId },
      });
    });

    it('should throw NotFoundException when template not found', async () => {
      const templateId = 'non-existent-id';

      mockPrismaService.template.findUnique.mockResolvedValue(null);

      await expect(service.findOne(templateId)).rejects.toThrow(
        NotFoundException,
      );
      await expect(service.findOne(templateId)).rejects.toThrow(
        `Template with ID ${templateId} not found`,
      );
    });
  });

  describe('create', () => {
    it('should create a template with version 1 when no existing template', async () => {
      const createDto: CreateTemplateDto = {
        name: 'welcome-email',
        language: 'en',
        subject: 'Welcome {{name}}',
        html_body: '<h1>Welcome {{name}}</h1>',
        text_body: 'Welcome {{name}}',
        variables: ['name'],
      };

      const createdTemplate = {
        id: 'new-template-id',
        ...createDto,
        language: 'en',
        version: 1,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      mockPrismaService.template.findFirst.mockResolvedValue(null);
      mockPrismaService.template.create.mockResolvedValue(createdTemplate);

      const result = await service.create(createDto);

      expect(result).toEqual(createdTemplate);
      expect(mockPrismaService.template.findFirst).toHaveBeenCalledWith({
        where: { name: 'welcome-email', language: 'en' },
        orderBy: { version: 'desc' },
      });
      expect(mockPrismaService.template.create).toHaveBeenCalledWith({
        data: {
          ...createDto,
          language: 'en',
          version: 1,
          is_active: true,
        },
      });
    });

    it('should increment version when template with same name and language exists', async () => {
      const createDto: CreateTemplateDto = {
        name: 'welcome-email',
        language: 'en',
        subject: 'Welcome {{name}}',
        html_body: '<h1>Welcome {{name}}</h1>',
        variables: ['name'],
      };

      const existingTemplate = {
        id: 'existing-id',
        name: 'welcome-email',
        language: 'en',
        version: 2,
      };

      const createdTemplate = {
        id: 'new-template-id',
        ...createDto,
        language: 'en',
        version: 3,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      mockPrismaService.template.findFirst.mockResolvedValue(existingTemplate);
      mockPrismaService.template.create.mockResolvedValue(createdTemplate);

      const result = await service.create(createDto);

      expect(result.version).toBe(3);
      expect(mockPrismaService.template.create).toHaveBeenCalledWith({
        data: {
          ...createDto,
          language: 'en',
          version: 3,
          is_active: true,
        },
      });
    });

    it('should default language to "en" when not provided', async () => {
      const createDto: CreateTemplateDto = {
        name: 'welcome-email',
        subject: 'Welcome {{name}}',
        html_body: '<h1>Welcome {{name}}</h1>',
        variables: ['name'],
      };

      const createdTemplate = {
        id: 'new-template-id',
        ...createDto,
        language: 'en',
        version: 1,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      mockPrismaService.template.findFirst.mockResolvedValue(null);
      mockPrismaService.template.create.mockResolvedValue(createdTemplate);

      await service.create(createDto);

      expect(mockPrismaService.template.create).toHaveBeenCalledWith({
        data: {
          ...createDto,
          language: 'en',
          version: 1,
          is_active: true,
        },
      });
    });
  });

  describe('findAll', () => {
    it('should return paginated templates with default page and limit', async () => {
      const mockTemplates = [
        {
          id: '1',
          name: 'template-1',
          language: 'en',
          subject: 'Subject 1',
          html_body: 'Body 1',
          variables: [],
          version: 1,
          is_active: true,
          created_at: new Date(),
          updated_at: new Date(),
        },
        {
          id: '2',
          name: 'template-2',
          language: 'en',
          subject: 'Subject 2',
          html_body: 'Body 2',
          variables: [],
          version: 1,
          is_active: true,
          created_at: new Date(),
          updated_at: new Date(),
        },
      ];

      mockPrismaService.template.findMany.mockResolvedValue(mockTemplates);
      mockPrismaService.template.count.mockResolvedValue(2);

      const result = await service.findAll();

      expect(result.data).toEqual(mockTemplates);
      expect(result.meta.total).toBe(2);
      expect(result.meta.page).toBe(1);
      expect(result.meta.limit).toBe(10);
      expect(result.meta.total_pages).toBe(1);
      expect(result.meta.has_next).toBe(false);
      expect(result.meta.has_previous).toBe(false);
      expect(mockPrismaService.template.findMany).toHaveBeenCalledWith({
        skip: 0,
        take: 10,
        orderBy: { created_at: 'desc' },
      });
    });

    it('should return paginated templates with custom page and limit', async () => {
      const mockTemplates = [
        {
          id: '1',
          name: 'template-1',
          language: 'en',
          subject: 'Subject 1',
          html_body: 'Body 1',
          variables: [],
          version: 1,
          is_active: true,
          created_at: new Date(),
          updated_at: new Date(),
        },
      ];

      mockPrismaService.template.findMany.mockResolvedValue(mockTemplates);
      mockPrismaService.template.count.mockResolvedValue(15);

      const result = await service.findAll(2, 5);

      expect(result.data).toEqual(mockTemplates);
      expect(result.meta.total).toBe(15);
      expect(result.meta.page).toBe(2);
      expect(result.meta.limit).toBe(5);
      expect(result.meta.total_pages).toBe(3);
      expect(result.meta.has_next).toBe(true);
      expect(result.meta.has_previous).toBe(true);
      expect(mockPrismaService.template.findMany).toHaveBeenCalledWith({
        skip: 5,
        take: 5,
        orderBy: { created_at: 'desc' },
      });
    });

    it('should handle empty results', async () => {
      mockPrismaService.template.findMany.mockResolvedValue([]);
      mockPrismaService.template.count.mockResolvedValue(0);

      const result = await service.findAll();

      expect(result.data).toEqual([]);
      expect(result.meta.total).toBe(0);
      expect(result.meta.total_pages).toBe(0);
    });
  });

  describe('render', () => {
    it('should render template with all variables replaced', async () => {
      const templateId = 'template-id';
      const mockTemplate = {
        id: templateId,
        name: 'welcome-email',
        language: 'en',
        subject: 'Welcome {{name}} to {{platform}}',
        html_body:
          '<h1>Welcome {{name}} to {{platform}}</h1><p>Your code is {{code}}</p>',
        text_body: 'Welcome {{name}} to {{platform}}. Your code is {{code}}',
        variables: ['name', 'platform', 'code'],
        version: 1,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      const renderDto: RenderTemplateDto = {
        template_id: templateId,
        variables: {
          name: 'John',
          platform: 'HNG',
          code: 'ABC123',
        },
      };

      mockPrismaService.template.findUnique.mockResolvedValue(mockTemplate);

      const result = await service.render(renderDto);

      expect(result.subject).toBe('Welcome John to HNG');
      expect(result.html_body).toBe(
        '<h1>Welcome John to HNG</h1><p>Your code is ABC123</p>',
      );
      expect(result.text_body).toBe('Welcome John to HNG. Your code is ABC123');
      expect(mockPrismaService.template.findUnique).toHaveBeenCalledWith({
        where: { id: templateId, is_active: true },
      });
    });

    it('should render template without text_body when not provided', async () => {
      const templateId = 'template-id';
      const mockTemplate = {
        id: templateId,
        name: 'welcome-email',
        language: 'en',
        subject: 'Welcome {{name}}',
        html_body: '<h1>Welcome {{name}}</h1>',
        text_body: null,
        variables: ['name'],
        version: 1,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      const renderDto: RenderTemplateDto = {
        template_id: templateId,
        variables: {
          name: 'John',
        },
      };

      mockPrismaService.template.findUnique.mockResolvedValue(mockTemplate);

      const result = await service.render(renderDto);

      expect(result.subject).toBe('Welcome John');
      expect(result.html_body).toBe('<h1>Welcome John</h1>');
      expect(result.text_body).toBeUndefined();
    });

    it('should throw NotFoundException when template not found', async () => {
      const renderDto: RenderTemplateDto = {
        template_id: 'non-existent-id',
        variables: {},
      };

      mockPrismaService.template.findUnique.mockResolvedValue(null);

      await expect(service.render(renderDto)).rejects.toThrow(
        NotFoundException,
      );
      await expect(service.render(renderDto)).rejects.toThrow(
        'Active template not found',
      );
    });

    it('should throw NotFoundException when template is inactive', async () => {
      const templateId = 'inactive-template-id';

      const renderDto: RenderTemplateDto = {
        template_id: templateId,
        variables: { name: 'John' },
      };

      // Since we filter by is_active: true, it won't find inactive templates
      mockPrismaService.template.findUnique.mockResolvedValue(null);

      await expect(service.render(renderDto)).rejects.toThrow(
        NotFoundException,
      );
    });

    it('should throw BadRequestException when required variables are missing', async () => {
      const templateId = 'template-id';
      const mockTemplate = {
        id: templateId,
        name: 'welcome-email',
        language: 'en',
        subject: 'Welcome {{name}} to {{platform}}',
        html_body: '<h1>Welcome {{name}} to {{platform}}</h1>',
        variables: ['name', 'platform', 'code'],
        version: 1,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      const renderDto: RenderTemplateDto = {
        template_id: templateId,
        variables: {
          name: 'John',
          // Missing 'platform' and 'code'
        },
      };

      mockPrismaService.template.findUnique.mockResolvedValue(mockTemplate);

      await expect(service.render(renderDto)).rejects.toThrow(
        BadRequestException,
      );
      await expect(service.render(renderDto)).rejects.toThrow(
        'Missing variables: platform, code',
      );
    });

    it('should handle multiple occurrences of the same variable', async () => {
      const templateId = 'template-id';
      const mockTemplate = {
        id: templateId,
        name: 'welcome-email',
        language: 'en',
        subject: 'Welcome {{name}}, {{name}}!',
        html_body: '<h1>Welcome {{name}}, {{name}}!</h1>',
        variables: ['name'],
        version: 1,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      const renderDto: RenderTemplateDto = {
        template_id: templateId,
        variables: {
          name: 'John',
        },
      };

      mockPrismaService.template.findUnique.mockResolvedValue(mockTemplate);

      const result = await service.render(renderDto);

      expect(result.subject).toBe('Welcome John, John!');
      expect(result.html_body).toBe('<h1>Welcome John, John!</h1>');
    });

    it('should handle extra variables that are not in template', async () => {
      const templateId = 'template-id';
      const mockTemplate = {
        id: templateId,
        name: 'welcome-email',
        language: 'en',
        subject: 'Welcome {{name}}',
        html_body: '<h1>Welcome {{name}}</h1>',
        variables: ['name'],
        version: 1,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
      };

      const renderDto: RenderTemplateDto = {
        template_id: templateId,
        variables: {
          name: 'John',
          extra: 'This should be ignored',
        },
      };

      mockPrismaService.template.findUnique.mockResolvedValue(mockTemplate);

      const result = await service.render(renderDto);

      expect(result.subject).toBe('Welcome John');
      expect(result.html_body).toBe('<h1>Welcome John</h1>');
    });
  });
});
