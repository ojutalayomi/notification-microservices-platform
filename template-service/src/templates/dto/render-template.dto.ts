import { IsString, IsObject } from 'class-validator';

export class RenderTemplateDto {
  @IsString()
  template_id: string;

  @IsObject()
  variables: Record<string, string>;
}
