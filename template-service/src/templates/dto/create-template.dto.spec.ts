import { CreateTemplateDto } from './create-template.dto';

describe('CreateTemplateDto', () => {
  it('should be defined', () => {
    const dto = new CreateTemplateDto();
    expect(dto).toBeDefined();
  });

  it('should accept valid template data', () => {
    const dto = new CreateTemplateDto();
    dto.name = 'welcome-email';
    dto.language = 'en';
    dto.subject = 'Welcome {{name}}';
    dto.html_body = '<h1>Welcome {{name}}</h1>';
    dto.variables = ['name'];

    expect(dto.name).toBe('welcome-email');
    expect(dto.language).toBe('en');
    expect(dto.subject).toBe('Welcome {{name}}');
    expect(dto.html_body).toBe('<h1>Welcome {{name}}</h1>');
    expect(dto.variables).toEqual(['name']);
  });
});
